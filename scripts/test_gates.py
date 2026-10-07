#!/usr/bin/env python3
"""gate-selftest: each CI gate fails on a planted breach, and passes clean.

Written for 0021-PLAN step 6.2 (0021-MADR Z9): a gate that cannot fail is
not a gate. Each test copies the tree into a temporary directory (a shared
clone of the repository, so the release tags api-check reads are there, with
the working tree's files laid over it), plants one breach, runs the gate,
and requires exit 1, the code for a failed check. The clean test runs every
gate on an unplanted copy and requires exit 0. Nothing in the tree itself is
changed.

Usage:
  make gate-selftest
  python3 -B -m unittest scripts/test_gates.py

It runs every package's tests twice, for coverage-check, so it takes some
minutes.
"""
from __future__ import annotations

import re
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
GATES = {
    "dep-check": "scripts/check_deps.py",
    "generate-check": "scripts/check_generated.py",
    "parity-check": "scripts/check_parity_map.py",
    "coverage-check": "scripts/check_coverage.py",
    "api-check": "scripts/check_api.py",
    "records-check": "scripts/check_records.py",
}
GENERATE_LINE = re.compile(r"^//go:generate .*$", re.M)


def copy_tree(dest: Path) -> None:
    """A shared clone of the repository at dest, with the working tree's
    tracked and untracked (not ignored) files copied over it."""
    subprocess.run(["git", "clone", "--quiet", "--shared", str(ROOT), str(dest)], check=True)
    listed = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "-z"], cwd=ROOT,
                            capture_output=True, check=True).stdout
    for rel in filter(None, listed.decode().split("\0")):
        src = ROOT / rel
        if src.is_file():
            (dest / rel).parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, dest / rel)
    deleted = subprocess.run(["git", "ls-files", "-d", "-z"], cwd=ROOT, capture_output=True, check=True).stdout
    for rel in filter(None, deleted.decode().split("\0")):
        (dest / rel).unlink(missing_ok=True)


def run_gate(tree: Path, gate: str) -> subprocess.CompletedProcess[str]:
    args = [sys.executable, "-B", GATES[gate]]
    if gate == "parity-check":
        args.append("--require-equivalents")
    return subprocess.run(args, cwd=tree, capture_output=True, text=True, check=False)


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text(encoding="utf-8")
    if text.count(old) != 1:
        raise AssertionError(f"{path.name}: {old!r} found {text.count(old)} times")
    path.write_text(text.replace(old, new), encoding="utf-8")


class GateSelfTest(unittest.TestCase):
    """One planted breach per gate, then every gate on a clean copy."""

    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.tree = Path(self.tmp.name) / "tree"
        copy_tree(self.tree)

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def assertFails(self, gate: str, want: str) -> None:
        proc = run_gate(self.tree, gate)
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, 1, f"{gate} exit {proc.returncode}, want 1:\n{out}")
        self.assertIn(want, out, f"{gate} did not report {want!r}:\n{out}")

    def test_dep_check_test_only_module(self) -> None:
        """A module only a test imports, required in go.mod."""
        evil = self.tree / "evil"
        evil.mkdir()
        (evil / "go.mod").write_text("module example.com/evil\n\ngo 1.27.1\n")
        (evil / "evil.go").write_text("package evil\n\n// X is planted.\nconst X = 1\n")
        (self.tree / "internal/redact/zz_evil_test.go").write_text(
            'package redact\n\nimport (\n\t"testing"\n\n\t"example.com/evil"\n)\n\n'
            "func TestEvil(t *testing.T) { _ = evil.X }\n")
        with (self.tree / "go.mod").open("a") as f:
            f.write("\nrequire example.com/evil v0.0.0\n\nreplace example.com/evil => ./evil\n")
        self.assertFails("dep-check", "example.com/evil")

    def _generate_file(self) -> tuple[Path, str]:
        doc = self.tree / "llmprovider/internal/ownerperm/doc.go"
        line = GENERATE_LINE.search(doc.read_text(encoding="utf-8"))
        self.assertIsNotNone(line, "no //go:generate line in ownerperm/doc.go")
        return doc, line.group(0)

    def test_generate_check_joined_output(self) -> None:
        """A hand-edited file behind a -output=X directive."""
        doc, line = self._generate_file()
        replace_once(doc, line, line.replace("-output zsyscall_windows.go", "-output=zsyscall_windows.go"))
        with (self.tree / "llmprovider/internal/ownerperm/zsyscall_windows.go").open("a") as f:
            f.write("\n// planted hand edit\n")
        self.assertFails("generate-check", "differs from what its //go:generate line produces")

    def test_generate_check_no_output(self) -> None:
        """Directives that lost their -output check nothing, which fails. Every
        package that generates code loses it: ownerperm and, since 0026-MADR
        F13, filelock."""
        for pkg in ("ownerperm", "filelock"):
            doc = self.tree / f"llmprovider/internal/{pkg}/doc.go"
            line = GENERATE_LINE.search(doc.read_text(encoding="utf-8"))
            self.assertIsNotNone(line, f"no //go:generate line in {pkg}/doc.go")
            replace_once(doc, line.group(0), line.group(0).replace(" -output zsyscall_windows.go", ""))
        self.assertFails("generate-check", "0 generated files")

    def test_parity_check_placeholder_cell(self) -> None:
        """A mapping cell that names nothing."""
        guide = self.tree / "docs/guides/migrating-from-mcplib.md"
        text = guide.read_text(encoding="utf-8")
        row = re.search(r"^(\|\s*`llmprovider\.[^`]+`\s*\|)([^|]*)(\|.*)$", text, re.M)
        self.assertIsNotNone(row, "no identifier row in the guide")
        guide.write_text(text.replace(row.group(0), f"{row.group(1)} TBD {row.group(3)}", 1), encoding="utf-8")
        self.assertFails("parity-check", "names nothing that resolves")

    def test_coverage_check_floor_above_measure(self) -> None:
        """A floor above what the package measures."""
        floors = self.tree / "scripts/coverage-floors.txt"
        text, n = re.subn(r"^wizard(\s+)\d+\.\d", r"wizard\g<1>99.9", floors.read_text(encoding="utf-8"),
                          count=1, flags=re.M)
        self.assertEqual(n, 1, "no wizard floor in coverage-floors.txt")
        floors.write_text(text, encoding="utf-8")
        self.assertFails("coverage-check", "wizard")

    def test_api_check_incompatible_change(self) -> None:
        """An exported constant that becomes a variable."""
        replace_once(self.tree / "llmprovider/catalog/models_catalog.go", "const MaxListed = 6", "var MaxListed = 6")
        self.assertFails("api-check", "MaxListed")

    def test_records_check_plan_status(self) -> None:
        """A complete PLAN given a decision's status (0025-MADR)."""
        plan = self.tree / "docs/decisions/0024-PLAN-opencode-live-system-message-test.md"
        replace_once(plan, "status: complete\n", "status: accepted\n")
        self.assertFails("records-check", f"{plan.name}: PLAN status 'accepted'")

    def test_clean_copy_passes_every_gate(self) -> None:
        for gate in GATES:
            with self.subTest(gate=gate):
                proc = run_gate(self.tree, gate)
                self.assertEqual(proc.returncode, 0, f"{gate} exit {proc.returncode} on a clean copy:\n"
                                                     f"{proc.stdout}{proc.stderr}")


if __name__ == "__main__":
    unittest.main()
