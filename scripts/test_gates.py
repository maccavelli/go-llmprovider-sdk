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

import contextlib
import importlib.util
import io
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parent.parent
GATES = {
    "dep-check": "scripts/check_deps.py",
    "generate-check": "scripts/check_generated.py",
    "parity-check": "scripts/check_parity_map.py",
    "coverage-check": "scripts/check_coverage.py",
    "api-check": "scripts/check_api.py",
    "contract-check": "scripts/check_contract.py",
    "records-check": "scripts/check_records.py",
    "tables-check": "scripts/check_tables.py",
    "fixtures-check": "scripts/check_fixtures.py",
    "changelog-check": "scripts/check_changelog.py",
    "docs-check": "scripts/check_docs.py",
    "links-check": "scripts/check_links.py",
    "disclosure-check": "scripts/check_disclosure.py",
}
GENERATE_LINE = re.compile(r"^//go:generate .*$", re.M)
# A //go:generate line's -output flag and its file, in either form.
OUTPUT_FLAG = re.compile(r" -output(?:=| +)\S+")


def copy_tree(dest: Path, source: Path = ROOT) -> None:
    """A shared clone of source at dest, with source's working tree's
    tracked and untracked (not ignored) files copied over it."""
    subprocess.run(["git", "clone", "--quiet", "--shared", str(source), str(dest)], check=True)
    listed = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "-z"], cwd=source,
                            capture_output=True, check=True).stdout
    for rel in filter(None, listed.decode().split("\0")):
        src = source / rel
        if src.is_file():
            (dest / rel).parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, dest / rel)
    # A deletion not yet staged is in the index and gone from the working
    # tree (ls-files -d); one already staged is in neither, so only diff
    # against HEAD finds it (0026-MADR F66).
    for cmd in (["git", "ls-files", "-d", "-z"], ["git", "diff", "--name-only", "--diff-filter=D", "-z", "HEAD"]):
        deleted = subprocess.run(cmd, cwd=source, capture_output=True, check=True).stdout
        for rel in filter(None, deleted.decode().split("\0")):
            (dest / rel).unlink(missing_ok=True)
    # The copy starts with no contract tag: a case that needs one plants its
    # own, and the source's real authcontract/v* tags would collide with it
    # (0073-PLAN, Phase 4, the tag's CI).
    tags = subprocess.run(["git", "tag", "-l", "authcontract/v*"], cwd=dest, capture_output=True, text=True,
                          check=True).stdout.split()
    if tags:
        subprocess.run(["git", "tag", "-d", *tags], cwd=dest, capture_output=True, check=True)


def run_gate(tree: Path, gate: str, *extra: str) -> subprocess.CompletedProcess[str]:
    args = [sys.executable, "-B", GATES[gate], *extra]
    if gate == "parity-check":
        args.append("--require-equivalents")
    return subprocess.run(args, cwd=tree, capture_output=True, text=True, check=False)


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text(encoding="utf-8")
    if text.count(old) != 1:
        raise AssertionError(f"{path.name}: {old!r} found {text.count(old)} times")
    path.write_text(text.replace(old, new), encoding="utf-8")


class CopyTreeTest(unittest.TestCase):
    """copy_tree copies what will be committed (0026-MADR F66)."""

    def test_copy_tree_drops_staged_deletion(self) -> None:
        """A tracked file whose deletion is staged is not in the copy: it is
        neither in the working tree nor in the index, so `git ls-files -d`
        does not list it, and the clone's HEAD still holds it."""
        with tempfile.TemporaryDirectory() as tmp:
            source = Path(tmp) / "source"
            subprocess.run(["git", "clone", "--quiet", "--shared", str(ROOT), str(source)], check=True)
            gone = "llmprovider/catalog/model_matcher.go"
            subprocess.run(["git", "rm", "--quiet", gone], cwd=source, check=True)
            dest = Path(tmp) / "copy"
            copy_tree(dest, source)
            self.assertFalse((dest / gone).exists(), f"the copy still holds {gone}, whose deletion is staged")
            self.assertTrue((dest / "llmprovider/catalog/discovery.go").exists(), "the copy lost a kept file")

    def test_copy_tree_drops_contract_tags(self) -> None:
        """The copy holds no authcontract/v* tag even when the source does,
        so the contract cases' own tags do not collide with a real one
        (0073-PLAN, Phase 4, the tag's CI)."""
        with tempfile.TemporaryDirectory() as tmp:
            source = Path(tmp) / "source"
            subprocess.run(["git", "clone", "--quiet", "--shared", str(ROOT), str(source)], check=True)
            subprocess.run(["git", "tag", "authcontract/v0.0.1"], cwd=source, check=True, capture_output=True)
            dest = Path(tmp) / "copy"
            copy_tree(dest, source)
            tags = subprocess.run(["git", "tag", "-l", "authcontract/v*"], cwd=dest, capture_output=True,
                                  text=True, check=True).stdout.split()
            self.assertEqual(tags, [], f"the copy holds the source's contract tags {tags}")


class PrecheckTest(unittest.TestCase):
    """go-precheck.sh checks what a commit changes (0026-MADR F15)."""

    def test_precheck_deleted_file_checks_its_package(self) -> None:
        """Given only a deleted .go path whose deletion breaks its package,
        the precheck fails, rather than finding no Go files to check."""
        with tempfile.TemporaryDirectory() as tmp:
            tree = Path(tmp) / "tree"
            copy_tree(tree)
            gone = "llmprovider/capabilities.go"
            (tree / gone).unlink()
            proc = subprocess.run(["bash", "scripts/go-precheck.sh", gone], cwd=tree, capture_output=True, text=True,
                                  check=False, env={**os.environ, "GO_PRECHECK_SKIP_VULN": "1"})
            out = proc.stdout + proc.stderr
            self.assertEqual(proc.returncode, 1, f"go-precheck exit {proc.returncode}, want 1:\n{out[-2000:]}")
            self.assertIn("go vet:", out, f"go vet did not report the broken package:\n{out[-2000:]}")

    def run_nested_precheck(self, planted: str, go_mod_extra: str = "") -> str:
        """Plant authcontract/plant.go (and go_mod_extra, appended to
        authcontract/go.mod) on a copy of the tree, run the precheck on the
        new file, require exit 1, and return its output (0073-PLAN Phase 1a)."""
        with tempfile.TemporaryDirectory() as tmp:
            tree = Path(tmp) / "tree"
            copy_tree(tree)
            (tree / "authcontract/plant.go").write_text(planted, encoding="utf-8")
            if go_mod_extra:
                with (tree / "authcontract/go.mod").open("a", encoding="utf-8") as f:
                    f.write(go_mod_extra)
            proc = subprocess.run(["bash", "scripts/go-precheck.sh", "authcontract/plant.go"], cwd=tree,
                                  capture_output=True, text=True, check=False,
                                  env={**os.environ, "GO_PRECHECK_SKIP_VULN": "1"})
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, 1, f"go-precheck exit {proc.returncode}, want 1:\n{out[-2000:]}")
        return out

    def test_precheck_nested_module_vets_its_file(self) -> None:
        """A file of the nested module authcontract is vetted in that module,
        not refused from the root as a directory of another module
        (0073-MADR D8)."""
        out = self.run_nested_precheck(
            'package authcontract\n\nimport "fmt"\n\nfunc plant() { fmt.Printf("%d", "x") }\n')
        self.assertIn("go vet (authcontract):", out, f"go vet did not run in the nested module:\n{out[-2000:]}")
        self.assertIn("plant.go", out, f"go vet did not name the planted file:\n{out[-2000:]}")
        self.assertIn('arg "x" of wrong type string', out, f"go vet did not report the planted error:\n{out[-2000:]}")

    def test_precheck_nested_module_tidy(self) -> None:
        """A requirement of the nested module that nothing imports fails
        `go mod tidy -diff` in that module (0073-MADR D8)."""
        out = self.run_nested_precheck(
            "// Package authcontract is planted.\npackage authcontract\n\n// X is planted.\nconst X = 1\n",
            "\nrequire golang.org/x/sys v0.47.0\n")
        self.assertIn("go mod tidy -diff (authcontract):", out, f"tidy did not run in the nested module:\n{out[-2000:]}")
        self.assertIn("-require golang.org/x/sys v0.47.0", out, f"tidy did not name the requirement:\n{out[-2000:]}")


class GateSelfTest(unittest.TestCase):
    """One planted breach per gate, then every gate on a clean copy."""

    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.tree = Path(self.tmp.name) / "tree"
        copy_tree(self.tree)

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def assertFails(self, gate: str, want: str) -> None:
        if gate == "changelog-check":
            keep_contract_tags(self.tree)
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

    def test_dep_check_allowed_module_outside_wizard(self) -> None:
        """A module go.mod allows, imported by a package that may not import
        it: only the per-package check catches it (0026-MADR F16)."""
        (self.tree / "llmprovider/zz_term.go").write_text(
            'package llmprovider\n\nimport "golang.org/x/term"\n\n// zzTerm is planted.\nvar zzTerm = term.IsTerminal\n')
        self.assertFails("dep-check", "llmprovider depends on golang.org/x/term")

    def test_dep_check_external_test_package(self) -> None:
        """An external test package shares its package's allowance, and no
        more: llmprovider_test may not import golang.org/x/term, and is named
        as its package (0048-PLAN Phase 4)."""
        (self.tree / "llmprovider/zz_term_test.go").write_text(
            'package llmprovider_test\n\nimport (\n\t"testing"\n\n\t"golang.org/x/term"\n)\n\n'
            "func TestZZTerm(t *testing.T) { _ = term.IsTerminal }\n")
        self.assertFails("dep-check", "llmprovider depends on golang.org/x/term")

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
        directive generate-check finds loses it, in either form, so a package
        that generates code is covered without naming it here: ownerperm,
        filelock (0026-MADR F13) and authcontract's schema (0073-PLAN Phase 3,
        the owner's decision of 2026-11-12)."""
        listed = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "*.go"], cwd=self.tree,
                                capture_output=True, text=True, check=True).stdout.split()
        stripped = []
        for rel in listed:
            path = self.tree / rel
            text = path.read_text(encoding="utf-8")
            for line in GENERATE_LINE.findall(text):
                bare = OUTPUT_FLAG.sub("", line)
                if bare != line:
                    replace_once(path, line, bare)
                    stripped.append(rel)
        for want in ("llmprovider/internal/ownerperm/doc.go", "llmprovider/internal/filelock/doc.go",
                     "authcontract/doc.go"):
            self.assertIn(want, stripped, f"no //go:generate line with -output in {want}")
        self.assertFails("generate-check", "0 generated files")

    def test_parity_check_placeholder_cell(self) -> None:
        """A mapping cell that names nothing."""
        guide = self.tree / "docs/guides/migrating-from-mcplib.md"
        text = guide.read_text(encoding="utf-8")
        row = re.search(r"^(\|\s*`llmprovider\.[^`]+`\s*\|)([^|]*)(\|.*)$", text, re.M)
        self.assertIsNotNone(row, "no identifier row in the guide")
        guide.write_text(text.replace(row.group(0), f"{row.group(1)} TBD {row.group(3)}", 1), encoding="utf-8")
        self.assertFails("parity-check", "names nothing that resolves")

    def _plant_cell(self, cell: str) -> None:
        """Replace the guide's first identifier row's SDK cell with cell."""
        guide = self.tree / "docs/guides/migrating-from-mcplib.md"
        text = guide.read_text(encoding="utf-8")
        row = re.search(r"^(\|\s*`llmprovider\.[^`]+`\s*\|)([^|]*)(\|.*)$", text, re.M)
        self.assertIsNotNone(row, "no identifier row in the guide")
        guide.write_text(text.replace(row.group(0), f"{row.group(1)} {cell} {row.group(3)}", 1), encoding="utf-8")

    def test_parity_check_bare_missing_function(self) -> None:
        """A bare call names a function that does not exist, only a field of
        that name (Descriptor.StaticModels) (0026-MADR F14)."""
        self._plant_cell("`StaticModels(ProviderClaude)`")
        self.assertFails("parity-check", "`StaticModels(ProviderClaude)`")

    def test_parity_check_unexported_name(self) -> None:
        """A cell naming something the package does not export: only
        unresolved() catches it (0026-MADR F16)."""
        self._plant_cell("`catalog.SearchEverything`")
        self.assertFails("parity-check", "`catalog.SearchEverything`")

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

    def test_api_check_wizard_subtree(self) -> None:
        """A change under wizard, the second subtree apidiff runs on: an
        exported constant that loses its type (0074-PLAN Phase 4)."""
        replace_once(self.tree / "wizard/auth.go", 'CredAPIKey CredentialKind = "api_key"', 'CredAPIKey = "api_key"')
        self.assertFails("api-check", "CredAPIKey")

    def test_api_check_package_outside_the_subtrees(self) -> None:
        """A public package under neither subtree, which no apidiff run
        would compare (0074-PLAN Phase 4)."""
        planted = self.tree / "zzplanted"
        planted.mkdir()
        (planted / "plant.go").write_text("// Package zzplanted is planted.\npackage zzplanted\n\n// X is planted.\nconst X = 1\n")
        self.assertFails("api-check", "zzplanted")

    def test_api_check_accepts_the_0093_removals(self) -> None:
        """0093-MADR MH39 (D10, Q1): against v1.32.0, each of D10's eight
        removals is accepted by name, citing the record, and the gate passes;
        any other incompatible change still fails (the cases above). The copy
        loses one of the eight, catalog.OptionsFromEnv, where the tree still
        holds it, with the two imports only it used. The base is named,
        `--against v1.32.0`: with none, api-check compares with the latest v1
        tag, which is v1.33.0 from its push on, and so with itself (0093-MADR
        amendment A3)."""
        path = self.tree / "llmprovider/catalog/model_metadata.go"
        text = path.read_text(encoding="utf-8")
        start = text.find("// OptionsFromEnv returns")
        if start != -1:
            end = text.index("\n}\n", start) + len("\n}\n")
            text = text[:start] + text[end:]
            text = text.replace('\t"os"\n', "", 1).replace('\t"strconv"\n', "", 1)
            path.write_text(text, encoding="utf-8")
        proc = run_gate(self.tree, "api-check", "--against", "v1.32.0")
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, 0, f"api-check exit {proc.returncode}, want 0:\n{out}")
        self.assertIn("accepted (0093-MADR D10): ./llmprovider/catalog.OptionsFromEnv: removed", out)

    def test_api_check_against_takes_only_a_release_tag(self) -> None:
        """--against names a v1.X.Y release tag: a branch, an option-like
        value and an empty value are refused (0093-MADR amendment A3)."""
        for value, given in (("main", ("--against", "main")), ("--list", ("--against=--list",)),
                             ("", ("--against=",))):
            with self.subTest(value=value):
                proc = run_gate(self.tree, "api-check", *given)
                out = proc.stdout + proc.stderr
                self.assertNotEqual(proc.returncode, 0, f"--against {value!r} exit 0, want non-zero:\n{out}")
                self.assertIn(f"--against {value!r}: want a release tag v1.X.Y", out)

    def test_records_check_plan_status(self) -> None:
        """A complete PLAN given a decision's status (0025-MADR)."""
        plan = self.tree / "docs/decisions/0024-PLAN-opencode-live-system-message-test.md"
        replace_once(plan, "status: complete\n", "status: accepted\n")
        self.assertFails("records-check", f"{plan.name}: PLAN status 'accepted'")

    def test_records_check_record_outside_its_directories(self) -> None:
        """A record placed under docs/guides/ is reported, and its number is
        counted by --next, since AGENTS.md numbers records found anywhere
        under docs/ (0026-MADR F61)."""
        (self.tree / "docs/guides/0099-MADR-stray-record.md").write_text("# Stray\n", encoding="utf-8")
        self.assertFails("records-check", "docs/guides/0099-MADR-stray-record.md")
        proc = subprocess.run([sys.executable, "-B", GATES["records-check"], "--next"], cwd=self.tree,
                              capture_output=True, text=True, check=False)
        self.assertEqual(proc.stdout.strip(), "0100", f"--next printed {proc.stdout.strip()!r}, want 0100")

    def test_tables_check_architecture_stale(self) -> None:
        """An architecture row names a package its package does not import
        (0046-MADR D4). The `wizard` row is found by its first cell, so the
        plant holds however its import list reads (0087-PLAN, deviation of
        2026-11-22)."""
        path = self.tree / "docs/architecture.md"
        rows = [line for line in path.read_text(encoding="utf-8").splitlines() if line.startswith("| `wizard` |")]
        self.assertEqual(len(rows), 1, "architecture.md: want one `wizard` row")
        row = rows[0]
        self.assertTrue(row.endswith(" |"), f"architecture.md: the `wizard` row does not end its last cell: {row!r}")
        replace_once(path, row, row[: -len(" |")] + ", `internal/transport` |")
        self.assertFails("tables-check", "docs/architecture.md: wizard")

    def test_tables_check_r2_disallowed(self) -> None:
        """A package imports what its R2 row does not allow."""
        replace_once(self.tree / "docs/guides/api-standards.md", "`internal/catalogtool`, `internal/transport` |",
                     "`internal/catalogtool` |")
        self.assertFails("tables-check", "api-standards.md: llmprovider/internal/cmd/catalogs")

    def test_tables_check_package_without_row(self) -> None:
        """A package with no row in either table."""
        planted = self.tree / "llmprovider/internal/zzplanted"
        planted.mkdir()
        (planted / "zzplanted.go").write_text("// Package zzplanted is planted.\npackage zzplanted\n", encoding="utf-8")
        self.assertFails("tables-check", "docs/architecture.md: llmprovider/internal/zzplanted: no row")
        self.assertFails("tables-check", "docs/guides/api-standards.md: llmprovider/internal/zzplanted: no row")

    def test_tables_check_live_variable_unnamed(self) -> None:
        """A live variable a test reads, which AGENTS.md does not name."""
        replace_once(self.tree / "AGENTS.md", "`LLMPROVIDER_LIVE_CHATGPT_PLAN` (the ChatGPT plan sign-in),",
                     "the ChatGPT plan sign-in,")
        self.assertFails("tables-check", "AGENTS.md: LLMPROVIDER_LIVE_CHATGPT_PLAN")

    def test_tables_check_switch_without_record(self) -> None:
        """A record measurement switch that no record of its number names."""
        (self.tree / "llmprovider/live_zz_planted_test.go").write_text(
            "//go:build live_gateways\n\npackage llmprovider_test\n\n// LLMPROVIDER_LIVE_0045_PLANTED is planted.\n",
            encoding="utf-8")
        self.assertFails("tables-check", "LLMPROVIDER_LIVE_0045_PLANTED")

    def test_tables_check_test_variable_unnamed(self) -> None:
        """A test-only variable a test file holds as a string literal, which
        neither AGENTS.md nor a record of its number names; a record numbered
        9999 that names it admits it (0090-MADR T13)."""
        (self.tree / "llmprovider/zz_planted_0090_test.go").write_text(
            'package llmprovider\n\n// zzPlanted0090 is planted.\nconst zzPlanted0090 = "FOO_TEST_9999_X"\n',
            encoding="utf-8")
        self.assertFails("tables-check", "FOO_TEST_9999_X")
        (self.tree / "docs/decisions/9999-MADR-zz-planted.md").write_text(
            "# Planted\n\nThe child reads FOO_TEST_9999_X.\n", encoding="utf-8")
        proc = run_gate(self.tree, "tables-check")
        self.assertEqual(proc.returncode, 0, f"tables-check exit {proc.returncode}, want 0:\n{proc.stdout}{proc.stderr}")

    def test_tables_check_test_variable_named_unheld(self) -> None:
        """A *_TEST_* variable AGENTS.md names that no test file holds, as
        after its test is gone (0095-MADR FU18)."""
        with (self.tree / "AGENTS.md").open("a", encoding="utf-8") as fh:
            fh.write("\nPlanted: ZZ_TEST_PLANTED_GONE is read by a test that is gone.\n")
        self.assertFails("tables-check", "AGENTS.md: ZZ_TEST_PLANTED_GONE: named, and no test file holds it")

    def test_tables_check_test_variable_raw_literal(self) -> None:
        """A test-only variable held as a raw (backquoted) string literal is
        read as an interpreted one is (0095-MADR FU19)."""
        (self.tree / "llmprovider/zz_planted_0095_test.go").write_text(
            "package llmprovider\n\n// zzPlanted0095 is planted.\nconst zzPlanted0095 = `FOO_TEST_RAW_X`\n",
            encoding="utf-8")
        self.assertFails("tables-check", "FOO_TEST_RAW_X")

    # 0047-PLAN Phase 1 step 6: each value below is made up; none may be printed.
    PLANT_FIXTURE = "llmprovider/catalog/testdata/zz_0047_plant.json"

    def assertFixtureCaught(self, value: str, kind: str) -> None:
        (self.tree / self.PLANT_FIXTURE).write_text('{"field": "' + value + '"}\n', encoding="utf-8")
        proc = run_gate(self.tree, "fixtures-check")
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, 1, f"fixtures-check exit {proc.returncode}, want 1:\n{out}")
        self.assertIn(f"{self.PLANT_FIXTURE}:1:", out)
        self.assertIn(f" {kind} ", out)
        self.assertNotIn(value, out, "fixtures-check printed the value it found")

    def test_fixtures_check_address(self) -> None:
        self.assertFixtureCaught("jane.q.public@acme-widgets.io", "email")

    def test_fixtures_check_account_id(self) -> None:
        self.assertFixtureCaught("org-Zq7Rb2Lm9Xc4Vt8N", "account-id")

    def test_fixtures_check_jwt(self) -> None:
        self.assertFixtureCaught("eyJhbGciOiJub25lIn0.eyJzdWIiOiJwbGFudGVkLTAwNDcifQ.c2ln", "jwt")

    def test_fixtures_check_workspace_id(self) -> None:
        """An Anthropic workspace id (0064-MADR, amendment 2026-10-27)."""
        self.assertFixtureCaught("wrkspc_01Zq7Rb2Lm9Xc4Vt8N", "workspace-id")

    def test_fixtures_check_stale_baseline_line(self) -> None:
        baseline = self.tree / "scripts/fixtures-baseline.txt"
        stale = "llmprovider/catalog/testdata/ranking-2026-10-12/together.json uuid 0047deadbeef"
        baseline.write_text(baseline.read_text(encoding="utf-8") + stale + "\n", encoding="utf-8")
        self.assertFails("fixtures-check", f"stale, no hit matches: {stale}")

    def test_fixtures_check_placeholders_pass(self) -> None:
        (self.tree / self.PLANT_FIXTURE).write_text(
            '{"safety_identifier": "user-REDACTED", "prompt_cache_key": "00000000-0000-0000-0000-000000000000",'
            ' "email": "user@example.com", "note": "user_message", "home": "/home/<user>/x",'
            ' "workspace": "wrkspc_REDACTED"}\n', encoding="utf-8")
        proc = run_gate(self.tree, "fixtures-check")
        self.assertEqual(proc.returncode, 0, f"placeholders were reported:\n{proc.stdout}{proc.stderr}")

    # 0048-PLAN Phase 6: the API audit, R50 to R52, on the copy.
    def test_docs_check_concurrency_sentence_removed(self) -> None:
        """An exported type whose doc no longer says whether it is safe for
        concurrent use."""
        replace_once(self.tree / "llmprovider/auth/oauth_device.go",
                     "// so one surface can show the code while another waits. It is safe for\n"
                     "// concurrent use: Cancel, and any number of Waits, may run at once.\n",
                     "// so one surface can show the code while another waits.\n")
        self.assertFails("docs-check", "R51 type DeviceLogin does not say whether it is safe for concurrent use")

    def test_docs_check_error_sentence_removed(self) -> None:
        """A function returning an error whose doc no longer names one."""
        replace_once(self.tree / "llmprovider/token.go", " It never returns an error.", "")
        self.assertFails("docs-check", "R52 StaticToken.Token returns an error its doc does not name")

    def test_docs_check_example_removed(self) -> None:
        """A public package left without an example."""
        (self.tree / "llmprovider/llmtest/example_test.go").unlink()
        self.assertFails("docs-check", "R50 package llmprovider/llmtest has no Example function with an Output comment")

    # 0048-PLAN Phase 5: CHANGELOG.md against the release tags.
    def test_changelog_check_tag_without_section(self) -> None:
        """A release tag with no section."""
        subprocess.run(["git", "tag", "v1.99.0"], cwd=self.tree, check=True, capture_output=True)
        self.assertFails("changelog-check", "v1.99.0 has no section")

    def test_changelog_check_contract_tag_without_section(self) -> None:
        """The nested module's tag with no section in its own changelog; the
        problem names the full tag (0073-PLAN Phase 2). The copy's v0.1.0
        section, which 0073-PLAN Phase 4 wrote, is removed first."""
        replace_once(self.tree / "authcontract/CHANGELOG.md", "## [v0.1.0] - 2026-11-12\n\n", "")
        subprocess.run(["git", "tag", "authcontract/v0.1.0"], cwd=self.tree, check=True, capture_output=True)
        self.assertFails("changelog-check", "authcontract/v0.1.0 has no section")

    def test_changelog_check_contract_wrong_date(self) -> None:
        """The nested module's section dated other than its tag's commit, by
        the rules of the root's."""
        subprocess.run(["git", "tag", "authcontract/v0.1.0"], cwd=self.tree, check=True, capture_output=True)
        replace_once(self.tree / "authcontract/CHANGELOG.md", "## [v0.1.0] - 2026-11-12\n",
                     "## [v0.1.0] - 2001-01-01\n")
        self.assertFails("changelog-check", "authcontract/v0.1.0 is dated 2001-01-01; its tag's commit is ")

    def test_changelog_check_out_of_order(self) -> None:
        """Two sections out of order, each with its own tag's date."""
        log = self.tree / "CHANGELOG.md"
        replace_once(log, "## [v1.14.0] - 2026-10-16", "## [v-swap]")
        replace_once(log, "## [v1.13.0] - 2026-10-13", "## [v1.14.0] - 2026-10-16")
        replace_once(log, "## [v-swap]", "## [v1.13.0] - 2026-10-13")
        self.assertFails("changelog-check", "sections out of order")

    def test_changelog_check_wrong_date(self) -> None:
        """A section dated other than its tag's commit."""
        replace_once(self.tree / "CHANGELOG.md", "## [v1.14.0] - 2026-10-16", "## [v1.14.0] - 2026-10-15")
        self.assertFails("changelog-check", "v1.14.0 is dated 2026-10-15; its tag's commit is 2026-10-16")

    def test_changelog_check_untagged_old_section(self) -> None:
        """A section with no tag, older than the newest tag: only the release
        being prepared, newer than every tag, may have no tag yet."""
        replace_once(self.tree / "CHANGELOG.md", "## [v1.14.0] - 2026-10-16",
                     "## [v1.14.1] - 2026-10-17\n\n### Fixed\n\n- Planted.\n\n## [v1.14.0] - 2026-10-16")
        self.assertFails("changelog-check", "v1.14.1 has a section and no tag")

    def changelog_section(self, *args: str) -> subprocess.CompletedProcess[str]:
        """check_changelog.py --section, in the copy, without CI's
        GITHUB_REPOSITORY, which --repo defaults to."""
        env = {k: v for k, v in os.environ.items() if k != "GITHUB_REPOSITORY"}
        return subprocess.run([sys.executable, "-B", GATES["changelog-check"], "--section", *args],
                              cwd=self.tree, env=env, capture_output=True, text=True, check=False)

    def test_changelog_section_of_unknown_tag(self) -> None:
        """--section of a tag the file has no section for exits 1."""
        proc = self.changelog_section("v0.0.9")
        self.assertEqual(proc.returncode, 1, f"--section v0.0.9 exit {proc.returncode}, want 1:\n{proc.stdout}{proc.stderr}")
        self.assertIn("no section for v0.0.9", proc.stderr)

    def test_changelog_section_prints_the_body(self) -> None:
        """--section prints its section's body: no heading, no comparison link."""
        proc = self.changelog_section("v1.0.0")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("The v1 API that", proc.stdout)
        self.assertNotIn("## [", proc.stdout)
        self.assertNotIn("]: https://", proc.stdout)

    def test_changelog_section_links_at_the_tag(self) -> None:
        """With --repo, a relative link names its file at the tag, its
        fragment kept, and an absolute link is kept; without it, the links are
        as written (0048-PLAN Phase 5 step 7)."""
        replace_once(self.tree / "CHANGELOG.md", "  [0042-MADR](docs/decisions/0042-MADR-ollama-models-that-write-text.md)\n",
                     "  [0042-MADR](docs/decisions/0042-MADR-ollama-models-that-write-text.md#d2)\n"
                     "  ([the form](https://keepachangelog.com/en/1.1.0/))\n")
        proc = self.changelog_section("v1.14.0", "--repo", "owner/name")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("](https://github.com/owner/name/blob/v1.14.0/docs/decisions/"
                      "0042-MADR-ollama-models-that-write-text.md#d2)", proc.stdout)
        self.assertIn("](https://keepachangelog.com/en/1.1.0/)", proc.stdout)
        self.assertNotIn("](docs/", proc.stdout)
        plain = self.changelog_section("v1.14.0")
        self.assertIn("](docs/decisions/0042-MADR-ollama-models-that-write-text.md#d2)", plain.stdout)

    def test_changelog_section_link_missing_at_the_tag(self) -> None:
        """With --repo, a link to a file the tag lacks fails --section."""
        replace_once(self.tree / "CHANGELOG.md", "(docs/decisions/0042-MADR-ollama-models-that-write-text.md)",
                     "(docs/decisions/9999-MADR-missing.md)")
        proc = self.changelog_section("v1.14.0", "--repo", "owner/name")
        self.assertEqual(proc.returncode, 1, f"exit {proc.returncode}, want 1:\n{proc.stdout}{proc.stderr}")
        self.assertIn("v1.14.0's section links docs/decisions/9999-MADR-missing.md, which v1.14.0 lacks", proc.stderr)

    # 0094-PLAN Phase 1: a released section, and its tag's compare link, are
    # compared with their text in the first tag's file that carries them
    # (0094-MADR D1 to D6). v1.0.0 to v1.15.0 have no CHANGELOG.md at their own
    # tags, so their baseline is v1.16.0's file; v1.24.0's is v1.24.1's. No
    # case names the newest tag or reads [Unreleased]'s entries; a case that
    # tags v99.0.0 or adds a section calls close_window first (RS17, RS33).
    V1_14_LAST = "  for `/v1/v1/models` and returned the static list.\n"
    V1_14_LINK = "[v1.14.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.13.0...v1.14.0\n"
    PLANTED = "### Fixed\n\n- Planted 0094."

    def changelog(self, code: int, *wants: str, absent: tuple[str, ...] = (), contract_tags: bool = True) -> str:
        """Run changelog-check with no argument, as make does; require code,
        each of wants in its output, and none of absent. The copy first gets
        the source's authcontract/v* tags back (RS34), unless contract_tags is
        false."""
        if contract_tags:
            keep_contract_tags(self.tree)
        proc = run_gate(self.tree, "changelog-check")
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, code, f"changelog-check exit {proc.returncode}, want {code}:\n{out}")
        for want in wants:
            self.assertIn(want, out, f"changelog-check did not report {want!r}:\n{out}")
        for not_want in absent:
            self.assertNotIn(not_want, out, f"changelog-check reported {not_want!r}:\n{out}")
        return out

    def line_of(self, text: str) -> int:
        """The 1-based line of CHANGELOG.md that starts text, which occurs once."""
        log = (self.tree / "CHANGELOG.md").read_text(encoding="utf-8")
        self.assertEqual(log.count(text), 1, f"{text!r} is not in CHANGELOG.md once")
        return log[:log.index(text)].count("\n") + 1

    def test_changelog_check_released_entry_added(self) -> None:
        """RS1, RS15: an entry added to v1.14.0's body fails, naming the
        heading's line, the baseline (v1.16.0, D1) and the first line that
        differs."""
        log = self.tree / "CHANGELOG.md"
        replace_once(log, self.V1_14_LAST, self.V1_14_LAST + "- Planted 0094: an entry in a released section.\n")
        heading, planted = self.line_of("## [v1.14.0] - "), self.line_of("- Planted 0094: an entry")
        self.changelog(1, f"line {heading}: v1.14.0's section differs from its text at v1.16.0, first at line {planted}")

    def test_changelog_check_released_backfilled_section(self) -> None:
        """RS2: v1.1.0, which no file at its own tag carries, is compared with
        v1.16.0's."""
        replace_once(self.tree / "CHANGELOG.md", "- Kilo accepts device-login sessions.\n",
                     "- Kilo accepts device-login sessions, planted 0094.\n")
        self.changelog(1, "v1.1.0's section differs from its text at v1.16.0")

    def test_changelog_check_released_late_section(self) -> None:
        """RS2: v1.24.0, whose own tag's file holds it under [Unreleased], is
        compared with v1.24.1's."""
        replace_once(self.tree / "CHANGELOG.md", "\n\n## [v1.23.0] - 2026-10-26",
                     "\n- Planted 0094 in v1.24.0.\n\n## [v1.23.0] - 2026-10-26")
        self.changelog(1, "v1.24.0's section differs from its text at v1.24.1")

    def test_changelog_check_released_carried_by_version(self) -> None:
        """RS2: a tag's file carries a section by its version alone: v99.0.0
        tagged at a file whose heading has no date is its baseline."""
        close_window(self.tree)
        add_section(self.tree, "## [v99.0.0]", self.PLANTED)
        tag_dated(self.tree, "v99.0.0", "2099-01-01")
        replace_once(self.tree / "CHANGELOG.md", "## [v99.0.0]\n", "## [v99.0.0] - 2099-01-01\n")
        replace_once(self.tree / "CHANGELOG.md", "- Planted 0094.", "- Planted 0094, edited.")
        self.changelog(1, "v99.0.0's section differs from its text at v99.0.0")

    def test_changelog_check_released_newest_tagged(self) -> None:
        """RS3, 0092-MADR ST27's shape: an entry put into the newest tagged
        section, found at run time, not named. A tag made before its date step
        carries no such section and is passed over (RS35)."""
        log = self.tree / "CHANGELOG.md"
        newest = next((m for m in DATED_RELEASE.finditer(log.read_text(encoding="utf-8"))
                       if tag_exists(self.tree, m.group(1)) and carried_at(self.tree, m.group(1))), None)
        self.assertIsNotNone(newest, "no release section has a tag")
        heading = newest.group(0)
        replace_once(log, heading + "\n", heading + "\n\n### Fixed\n\n- Planted 0094 in the newest tagged section.\n")
        self.changelog(1, f"{newest.group(1)}'s section differs from its text at ")

    def test_changelog_check_released_no_baseline(self) -> None:
        """RS4, RS15: a released section no file carries passes when its tag
        is the newest, counted as having no released text, and fails when an
        older tag."""
        close_window(self.tree)
        tag_dated(self.tree, "v99.0.0", "2099-01-01")
        v99 = add_section(self.tree, "## [v99.0.0] - 2099-01-01", self.PLANTED)
        self.changelog(0, "1 with no released text")
        log = self.tree / "CHANGELOG.md"
        replace_once(log, v99, "")
        v991 = add_section(self.tree, "## [v99.0.1] - 2099-01-02", "### Fixed\n\n- Planted 0094, v99.0.1.")
        tag_dated(self.tree, "v99.0.1", "2099-01-02")
        replace_once(log, v991, v991 + v99)
        self.changelog(1, "v99.0.0's section has no released text: no tag's file carries it")

    def test_changelog_check_unreleased_and_release_commit_pass(self) -> None:
        """RS5, RS20: [Unreleased], one untagged section newer than every tag,
        and the preamble are not compared."""
        close_window(self.tree)
        log = self.tree / "CHANGELOG.md"
        replace_once(log, "## [Unreleased]\n", "## [Unreleased]\n\n- Planted 0094 under [Unreleased].\n")
        self.changelog(0)
        add_section(self.tree, "## [v99.0.0] - 2099-01-01", self.PLANTED)
        self.changelog(0)
        replace_once(log, "# Changelog\n", "# Changelog\n\nPlanted 0094 in the preamble.\n")
        self.changelog(0)

    def test_changelog_check_released_after_its_tag(self) -> None:
        """RS6: a section equals its text at its own tag; an edit after the
        tag fails, naming it."""
        close_window(self.tree)
        add_section(self.tree, "## [v99.0.0] - 2099-01-01", self.PLANTED)
        tag_dated(self.tree, "v99.0.0", "2099-01-01")
        self.changelog(0)
        replace_once(self.tree / "CHANGELOG.md", "- Planted 0094.", "- Planted 0094.\n- Added after the tag.")
        self.changelog(1, "v99.0.0's section differs from its text at v99.0.0")

    def test_changelog_check_released_whitespace_passes(self) -> None:
        """RS7: trailing spaces on a released line and a blank line at a
        released section's end are no difference."""
        log = self.tree / "CHANGELOG.md"
        replace_once(log, self.V1_14_LAST, self.V1_14_LAST.replace("list.\n", "list.   \n"))
        replace_once(log, "Phase 9.\n\n## [v1.13.0] - 2026-10-13", "Phase 9.\n\n\n## [v1.13.0] - 2026-10-13")
        self.changelog(0)

    def test_changelog_check_released_rewrap(self) -> None:
        """RS8: two lines of a released entry joined into one differ."""
        replace_once(self.tree / "CHANGELOG.md", "Since `v1.0.0` it had asked\n  for `/v1/v1/models`",
                     "Since `v1.0.0` it had asked for `/v1/v1/models`")
        self.changelog(1, "v1.14.0's section differs from its text at v1.16.0")

    def test_changelog_check_released_link_changed(self) -> None:
        """RS9 (a): a released tag's compare link changed, then removed."""
        log = self.tree / "CHANGELOG.md"
        changed = self.V1_14_LINK.replace("v1.13.0...", "v1.12.0...")
        replace_once(log, self.V1_14_LINK, changed)
        self.changelog(1, f"line {self.line_of(changed)}: v1.14.0's compare link differs from its line at v1.16.0")
        replace_once(log, changed, "")
        self.changelog(1, "v1.14.0's compare link differs from its line at v1.16.0")

    def test_changelog_check_released_last_section(self) -> None:
        """RS9 (b): the last section, v1.0.0, which the compare links follow,
        is compared."""
        replace_once(self.tree / "CHANGELOG.md", "fails on an incompatible change.\n",
                     "fails on an incompatible change.\n- Planted 0094 in the last section.\n")
        self.changelog(1, "v1.0.0's section differs from its text at v1.16.0")

    def test_changelog_check_unreleased_link_passes(self) -> None:
        """RS9 (c): the [Unreleased] link changed and a new tag's link added."""
        log = self.tree / "CHANGELOG.md"
        text = log.read_text(encoding="utf-8")
        link = re.search(r"^\[Unreleased\]: \S+\n", text, re.M)
        self.assertIsNotNone(link, "CHANGELOG.md has no [Unreleased] link")
        replace_once(log, link.group(0), "[Unreleased]: https://example.invalid/planted-0094...HEAD\n"
                                         "[v99.0.0]: https://example.invalid/planted-0094\n")
        self.changelog(0)

    def test_changelog_check_released_contract_section(self) -> None:
        """RS10: the nested module's section, compared by its own tags, the
        problem naming its file and full tag."""
        tag_dated(self.tree, "authcontract/v0.1.0", "2026-11-12")
        replace_once(self.tree / "authcontract/CHANGELOG.md", "- The evolution rules, in the package doc",
                     "- Planted 0094.\n- The evolution rules, in the package doc")
        self.changelog(1, "authcontract/CHANGELOG.md: line ",
                       "authcontract/v0.1.0's section differs from its text at authcontract/v0.1.0",
                       absent=("is dated",))

    def test_changelog_check_contract_untagged_not_compared(self) -> None:
        """RS10: with no authcontract/v* tag, as copy_tree leaves a copy (E24),
        the nested file's one section is not compared."""
        nested = self.tree / "authcontract/CHANGELOG.md"
        text = nested.read_text(encoding="utf-8")
        heads = list(DATED_RELEASE.finditer(text))
        # One section and no tag is this case's state; a later untagged nested
        # section beside it is RS11's missing tags (0094-PLAN D-4).
        nested.write_text(text[:heads[0].start()] + text[heads[-1].start():], encoding="utf-8")
        replace_once(nested, "- The evolution rules, in the package doc",
                     "- Planted 0094.\n- The evolution rules, in the package doc")
        self.changelog(0, contract_tags=False)

    def test_changelog_check_contract_date_step_passes(self) -> None:
        """RS34: the nested module's date step, a section untagged above its
        tagged ones, passes in a copy, as it does in CI. A date step the tree
        already holds, untagged in the source, is tagged first, as close_window
        does for the root's."""
        keep_contract_tags(self.tree)
        nested = self.tree / "authcontract/CHANGELOG.md"
        first = DATED_RELEASE.search(nested.read_text(encoding="utf-8"))
        if not tag_exists(ROOT, f"authcontract/{first.group(1)}"):
            tag_dated(self.tree, f"authcontract/{first.group(1)}", first.group(2))
        replace_once(nested, first.group(0) + "\n",
                     f"## [v0.99.0] - 2099-01-01\n\n### Added\n\n- Planted 0094.\n\n{first.group(0)}\n")
        self.changelog(0)

    def test_changelog_check_working_file_not_utf8(self) -> None:
        """RS36: a working changelog that is not UTF-8 is a tool error, for
        the gate and for --section."""
        for rel in ("CHANGELOG.md", "authcontract/CHANGELOG.md"):
            with self.subTest(file=rel):
                path = self.tree / rel
                original = path.read_bytes()
                path.write_bytes(original + b"\xff planted 0094\n")
                self.changelog(2, f"changelog-check: {rel} is not UTF-8")
                if rel == "CHANGELOG.md":
                    proc = self.changelog_section("v1.0.0")
                    self.assertEqual(proc.returncode, 2, f"--section exit {proc.returncode}, want 2:\n{proc.stderr}")
                    self.assertIn("changelog-check: CHANGELOG.md is not UTF-8", proc.stderr)
                path.write_bytes(original)

    def drop_tags(self, keep: re.Pattern[str] | None = None) -> None:
        """Delete the copy's v* tags, except those keep matches."""
        tags = subprocess.run(["git", "tag", "-l", "v*"], cwd=self.tree, capture_output=True, text=True,
                              check=True).stdout.split()
        gone = [t for t in tags if keep is None or not keep.match(t)]
        if gone:
            subprocess.run(["git", "tag", "-d", *gone], cwd=self.tree, capture_output=True, check=True)

    def test_changelog_check_missing_tags_is_a_tool_error(self) -> None:
        """RS11, RS15: two or more release sections untagged and newer than
        every tag: the clone lacks tags, exit 2."""
        want = ("release sections are newer than every v* tag; one can be the release being prepared; "
                "this clone lacks tags (CI checks out with fetch-depth: 0)")
        with self.subTest(case="no tags"):
            self.drop_tags()
            self.changelog(2, "changelog-check: CHANGELOG.md: ", want)
        self.tearDown()
        self.setUp()
        with self.subTest(case="tags from v1.20.0 on deleted"):
            self.drop_tags(re.compile(r"^v1\.(?:[0-9]|1[0-9])\.\d+$"))
            self.changelog(2, want)
        self.tearDown()
        self.setUp()
        with self.subTest(case="two untagged above a closed window"):
            close_window(self.tree)
            add_section(self.tree, "## [v99.0.0] - 2099-01-01", self.PLANTED)
            add_section(self.tree, "## [v99.0.1] - 2099-01-02", self.PLANTED)
            self.changelog(2, f"2 {want}")

    def test_changelog_check_shallow_is_a_tool_error(self) -> None:
        """RS12 (a): a shallow clone, exit 2."""
        head = subprocess.run(["git", "rev-parse", "HEAD"], cwd=self.tree, capture_output=True, text=True,
                              check=True).stdout
        (self.tree / ".git/shallow").write_text(head, encoding="utf-8")
        self.changelog(2, "shallow")

    def set_accepted(self, entries: str) -> None:
        """The copy's ACCEPTED, as Python source."""
        replace_once(self.tree / "scripts/check_changelog.py", "ACCEPTED: dict[str, tuple[str, str]] = {}",
                     f"ACCEPTED: dict[str, tuple[str, str]] = {entries}")

    def test_changelog_check_released_accepted_edit(self) -> None:
        """RS13, RS15: an ACCEPTED entry makes its reviewed commit the
        section's baseline, body and compare link; RS37: a stale entry is a
        problem."""
        close_window(self.tree)
        log = self.tree / "CHANGELOG.md"
        original = log.read_text(encoding="utf-8")
        replace_once(log, self.V1_14_LAST, self.V1_14_LAST + "- Fixed after release, planted 0094.\n")
        replace_once(log, self.V1_14_LINK, self.V1_14_LINK.replace("v1.13.0...", "v1.12.0..."))
        fixed = commit_tree(self.tree, move_head=False)
        script = self.tree / "scripts/check_changelog.py"
        unset = script.read_text(encoding="utf-8")
        with self.subTest(case="(a) accepted"):
            self.set_accepted(f'{{"v1.14.0": ("{fixed}", "0094-MADR plant")}}')
            self.changelog(0, f"accepted (0094-MADR plant): v1.14.0 at {fixed}", ", 1 accepted, ")
        with self.subTest(case="(b) a further edit"):
            replace_once(log, "- Fixed after release, planted 0094.\n", "- Fixed after release, planted 0094, again.\n")
            self.changelog(1, f"v1.14.0's section differs from its text at {fixed}")
        log.write_text(original.replace("## [v1.14.0] - 2026-10-16\n", "## [v1.13.9] - 2026-10-16\n"),
                       encoding="utf-8")
        carrierless = commit_tree(self.tree, move_head=False)
        log.write_text(original, encoding="utf-8")
        with self.subTest(case="(c) the commit does not carry the section"):
            script.write_text(unset, encoding="utf-8")
            self.set_accepted(f'{{"v1.14.0": ("{carrierless}", "0094-MADR plant")}}')
            self.changelog(1, f"v1.14.0's section has no text at {carrierless}")
        with self.subTest(case="(d) an absent commit"):
            script.write_text(unset, encoding="utf-8")
            self.set_accepted(f'{{"v1.14.0": ("{"0" * 40}", "0094-MADR plant")}}')
            self.changelog(2, "0" * 40)
        for case, entries in (("(e) not a full object name", '{"v1.14.0": ("main", "0094-MADR plant")}'),
                              ("(e) no record", f'{{"v1.14.0": ("{fixed}", "")}}')):
            with self.subTest(case=case):
                script.write_text(unset, encoding="utf-8")
                self.set_accepted(entries)
                self.changelog(2, "ACCEPTED")
        with self.subTest(case="(f) a tag that is not a release here"):
            script.write_text(unset, encoding="utf-8")
            self.set_accepted(f'{{"v1.99.0": ("{fixed}", "0094-MADR plant")}}')
            self.changelog(1, "ACCEPTED names v1.99.0, which is not a release tag in this clone")
        with self.subTest(case="(g) a tag with no section"):
            script.write_text(unset, encoding="utf-8")
            tag_dated(self.tree, "v99.0.0", "2099-01-01")
            self.set_accepted(f'{{"v99.0.0": ("{fixed}", "0094-MADR plant")}}')
            self.changelog(1, "ACCEPTED names v99.0.0, which has no section")

    def test_changelog_check_released_duplicate_section(self) -> None:
        """RS18: a released section repeated below itself, its copy's body
        edited: the existing problem, and the first occurrence compared."""
        replace_once(self.tree / "CHANGELOG.md", "## [v1.13.0] - 2026-10-13",
                     "## [v1.14.0] - 2026-10-16\n\n- Planted 0094 in a duplicate.\n\n## [v1.13.0] - 2026-10-13")
        self.changelog(1, "v1.14.0 has a section already", absent=("differs",))

    def test_changelog_check_released_tag_not_utf8(self) -> None:
        """RS21: a tag's file that is not UTF-8 is a tool error naming the tag.
        The gate first passes after close_window (0094-MADR A1), so a no-op
        close_window fails this case in the release window."""
        close_window(self.tree)
        self.changelog(0)
        log = self.tree / "CHANGELOG.md"
        original = log.read_bytes()
        log.write_bytes(original + b"\xff planted 0094\n")
        tag_dated(self.tree, "v99.0.0", "2099-01-01")
        log.write_bytes(original)
        add_section(self.tree, "## [v99.0.0] - 2099-01-01", self.PLANTED)
        self.changelog(2, "v99.0.0", "not UTF-8")

    def test_changelog_check_released_yanked(self) -> None:
        """RS29, RS19: a released heading may be marked [YANKED]; its date is
        still checked, its body still compared, and --section finds it."""
        log = self.tree / "CHANGELOG.md"
        with self.subTest(case="(a) marked"):
            replace_once(log, "## [v1.14.0] - 2026-10-16\n", "## [v1.14.0] - 2026-10-16 [YANKED]\n")
            self.changelog(0)
        with self.subTest(case="(d) --section"):
            proc = self.changelog_section("v1.14.0")
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertIn(self.V1_14_LAST.strip(), proc.stdout)
        with self.subTest(case="(c) its body edited"):
            replace_once(log, self.V1_14_LAST, self.V1_14_LAST + "- Planted 0094 in a yanked section.\n")
            self.changelog(1, "v1.14.0's section differs from its text at v1.16.0")
        with self.subTest(case="(b) the wrong date"):
            replace_once(log, "- Planted 0094 in a yanked section.\n", "")
            replace_once(log, "## [v1.14.0] - 2026-10-16 [YANKED]\n", "## [v1.14.0] - 2026-10-15 [YANKED]\n")
            self.changelog(1, "v1.14.0 is dated 2026-10-15; its tag's commit is 2026-10-16")

    def test_changelog_check_yanked_untagged(self) -> None:
        """RS30: [YANKED] on a section with no tag."""
        close_window(self.tree)
        add_section(self.tree, "## [v99.0.0] - 2099-01-01 [YANKED]", self.PLANTED)
        self.changelog(1, "v99.0.0 is marked [YANKED] and has no tag")

    def test_changelog_check_released_link_twice(self) -> None:
        """RS31: a second compare-link definition for a released label."""
        replace_once(self.tree / "CHANGELOG.md", self.V1_14_LINK, self.V1_14_LINK * 2)
        self.changelog(1, f"line {self.line_of(self.V1_14_LINK * 2) + 1}: a second compare link for v1.14.0")

    def test_changelog_check_prerelease_tag_ignored(self) -> None:
        """RS32: a tag not of the form vX.Y.Z is neither a release nor a
        baseline."""
        close_window(self.tree)
        add_section(self.tree, "## [v99.0.0] - 2099-01-01", self.PLANTED)
        tag_dated(self.tree, "v99.0.0-rc.1", "2099-01-01")
        replace_once(self.tree / "CHANGELOG.md", "- Planted 0094.", "- Planted 0094, after the candidate.")
        self.changelog(0)

    # 0063-PLAN Phase 1: the documentation gates both workflows run (0063-MADR
    # R1 to R3). Each planted home path is built at run time, so this file
    # holds none for disclosure-check to find.
    def test_links_check_broken_link(self) -> None:
        """A relative link to a file that does not exist."""
        with (self.tree / "README.md").open("a", encoding="utf-8") as f:
            f.write("\nSee [the plant](docs/guides/missing-0063.md).\n")
        self.assertFails("links-check", ": docs/guides/missing-0063.md")

    def lint_files(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run([sys.executable, "-B", GATES["links-check"], "--lint-files"], cwd=self.tree,
                              capture_output=True, text=True, check=False)

    def test_links_check_lint_files_skip_the_records(self) -> None:
        """markdownlint's list: a new Markdown file is in it, the records are
        not (0047-MADR A2)."""
        (self.tree / "docs/guides/zz-0063.md").write_text("# Planted\n", encoding="utf-8")
        proc = self.lint_files()
        self.assertEqual(proc.returncode, 0, proc.stderr)
        listed = proc.stdout.splitlines()
        self.assertIn("docs/guides/zz-0063.md", listed)
        self.assertIn("README.md", listed)
        self.assertEqual([f for f in listed if f.startswith(("docs/decisions/", "docs/reports/"))], [])

    def test_markdownlint_target_lints_the_list_and_stops_on_its_failure(self) -> None:
        """make markdownlint passes the list to the linter, and runs no linter
        when the list cannot be given: a file name with white space."""
        def lint() -> subprocess.CompletedProcess[str]:
            return subprocess.run(["make", "-s", "markdownlint", "MARKDOWNLINT=echo LINTED"], cwd=self.tree,
                                  capture_output=True, text=True, check=False)
        clean = lint()
        self.assertEqual(clean.returncode, 0, clean.stderr)
        self.assertIn("LINTED --no-globs", clean.stdout)
        self.assertIn(" README.md", clean.stdout)
        (self.tree / "docs/guides/zz 0063.md").write_text("# Planted\n", encoding="utf-8")
        spaced = lint()
        self.assertNotEqual(spaced.returncode, 0, f"make markdownlint passed:\n{spaced.stdout}{spaced.stderr}")
        self.assertNotIn("LINTED", spaced.stdout)
        self.assertIn("names with white space", spaced.stderr)

    def assertHomeCaught(self, line: str, kind: str) -> None:
        plant = self.tree / "zz-disclosure-0063.txt"
        plant.write_text(line + "\n", encoding="utf-8")
        proc = run_gate(self.tree, "disclosure-check")
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, 1, f"disclosure-check exit {proc.returncode}, want 1:\n{out}")
        self.assertIn(f"zz-disclosure-0063.txt:1: {kind}", out)
        self.assertNotIn("jdoe", out, "disclosure-check printed the path it found")

    def test_disclosure_check_unix_home(self) -> None:
        for home in ("Users", "home"):
            with self.subTest(home=home):
                self.assertHomeCaught("cloned at /" + home + "/jdoe/src", "a macOS or Linux home path")

    def test_disclosure_check_windows_home(self) -> None:
        for sep in ("\\", "\\\\", "/"):
            with self.subTest(sep=sep):
                self.assertHomeCaught("cloned at C:" + sep + "Users" + sep + "jdoe" + sep + "src",
                                      "a Windows home path")

    def test_disclosure_check_placeholders_pass(self) -> None:
        (self.tree / "zz-disclosure-0063.txt").write_text(
            "/home/<user>/src /Users/<user>/src C:\\Users\\<user>\\src C:/Users/<user>/src\n", encoding="utf-8")
        proc = run_gate(self.tree, "disclosure-check")
        self.assertEqual(proc.returncode, 0, f"placeholders were reported:\n{proc.stdout}{proc.stderr}")

    def test_disclosure_check_unreadable_file(self) -> None:
        """A file the check cannot read fails it, as a tool error."""
        if hasattr(os, "geteuid") and os.geteuid() == 0:
            self.skipTest("root reads a file whatever its mode")
        plant = self.tree / "zz-disclosure-0063.txt"
        plant.write_text("x\n", encoding="utf-8")
        plant.chmod(0)
        proc = run_gate(self.tree, "disclosure-check")
        self.assertEqual(proc.returncode, 2, f"disclosure-check exit {proc.returncode}, want 2:\n{proc.stdout}{proc.stderr}")
        self.assertIn("disclosure-check: ", proc.stderr)

    def test_clean_copy_passes_every_gate(self) -> None:
        for gate in GATES:
            with self.subTest(gate=gate):
                if gate == "changelog-check":
                    keep_contract_tags(self.tree)
                proc = run_gate(self.tree, gate)
                self.assertEqual(proc.returncode, 0, f"{gate} exit {proc.returncode} on a clean copy:\n"
                                                     f"{proc.stdout}{proc.stderr}")


def commit_tree(tree: Path, date: str | None = None, move_head: bool = True) -> str:
    """Commit tree's working tree with `git commit-tree`, so that no hook runs,
    as tools/_common.scratch_clone does, and return the commit; with date, its
    author and committer dates are noon UTC that day. The repository the copy
    was cloned from is never touched."""
    env = {**os.environ, "GIT_AUTHOR_NAME": "tools", "GIT_AUTHOR_EMAIL": "tools@invalid",
           "GIT_COMMITTER_NAME": "tools", "GIT_COMMITTER_EMAIL": "tools@invalid"}
    if date is not None:
        env["GIT_AUTHOR_DATE"] = env["GIT_COMMITTER_DATE"] = f"{date}T12:00:00+0000"

    def git(*args: str) -> str:
        return subprocess.run(["git", *args], cwd=tree, env=env, capture_output=True, text=True,
                              check=True).stdout.strip()

    git("add", "-A")
    commit = git("commit-tree", git("write-tree"), "-p", "HEAD", "-m", "scratch tag")
    if move_head:
        git("update-ref", "HEAD", commit)
    return commit


def tag_tree(tree: Path, tag: str, date: str | None = None) -> None:
    """Commit tree's working tree (commit_tree) and tag that commit in the copy."""
    commit = commit_tree(tree, date)
    subprocess.run(["git", "tag", tag, commit], cwd=tree, capture_output=True, check=True)


def tag_dated(tree: Path, tag: str, date: str) -> None:
    """tag_tree with the commit dated date, so a changelog heading can carry
    that date before its tag exists (0094-MADR, its state table)."""
    tag_tree(tree, tag, date)


def tag_exists(tree: Path, tag: str) -> bool:
    return subprocess.run(["git", "rev-parse", "-q", "--verify", f"refs/tags/{tag}"], cwd=tree,
                          capture_output=True, check=False).returncode == 0


def keep_contract_tags(tree: Path, source: Path = ROOT) -> None:
    """Give the copy back each authcontract/v* tag source holds, at its own
    object, which the shared clone holds, unless the copy holds that tag name
    already: a case that planted its own keeps it. copy_tree drops them (E24),
    and with none, the nested module's date step would read as missing tags
    (0094-MADR RS34)."""
    refs = subprocess.run(["git", "for-each-ref", "--format=%(refname:strip=2) %(objectname)", "refs/tags/authcontract/"],
                          cwd=source, capture_output=True, text=True, check=True).stdout.splitlines()
    for ref in refs:
        name, obj = ref.split()
        if not tag_exists(tree, name):
            subprocess.run(["git", "tag", name, obj], cwd=tree, capture_output=True, check=True)


def carried_at(tree: Path, tag: str) -> bool:
    """Whether tag's own CHANGELOG.md has a section for tag: false for a tag
    made before its date step (0094-MADR RS35)."""
    shown = subprocess.run(["git", "show", f"{tag}:CHANGELOG.md"], cwd=tree, capture_output=True, text=True,
                           check=False)
    return shown.returncode == 0 and any(line.startswith(f"## [{tag}]") for line in shown.stdout.splitlines())


# A dated release heading, as a test finds one; the gate's own form is
# check_changelog.RELEASE.
DATED_RELEASE = re.compile(r"^## \[(v\d+\.\d+\.\d+)\] - (\d{4}-\d{2}-\d{2})", re.M)


def close_window(tree: Path) -> None:
    """When the copy's first release section has no tag (the release being
    prepared, between its date step and its tag), tag it at a commit dated by
    its heading, so a case that tags v99.0.0 or adds a section holds in the
    release window too (0094-MADR RS17, RS33)."""
    first = DATED_RELEASE.search((tree / "CHANGELOG.md").read_text(encoding="utf-8"))
    if first is not None and not tag_exists(tree, first.group(1)):
        tag_dated(tree, first.group(1), first.group(2))


def add_section(tree: Path, heading: str, body: str) -> str:
    """Put a section above the copy's first dated release section, and return
    the text added."""
    path = tree / "CHANGELOG.md"
    text = path.read_text(encoding="utf-8")
    first = DATED_RELEASE.search(text)
    if first is None:
        raise AssertionError("CHANGELOG.md has no dated release section")
    added = f"{heading}\n\n{body}\n\n"
    path.write_text(text[:first.start()] + added + text[first.start():], encoding="utf-8")
    return added


# A schema as scripts/check_contract.py reads one, written by hand: the object
# Step with its url member and the open enum Paste. The cases that use it came
# before the generator of 0073-PLAN Phase 3; the cases named `_generated` run
# the generator instead.
PLANT_SCHEMA = {
    "$defs": {
        "Step": {"type": "object",
                 "properties": {"seq": {"type": "integer"}, "url": {"type": "string"}, "kind": {"type": "string"}},
                 "required": ["seq", "kind"]},
        "Paste": {"type": "string", "enum": ["", "url_or_code", "url_only", "code"], "x-open-enum": True},
    },
}


class ContractCheckTest(unittest.TestCase):
    """contract-check fails on each planted break of the nested module
    authcontract (0073-MADR D7; 0073-PLAN Phase 2). A plant that needs a tag
    tags the copy first, then breaks the tree."""

    TAG = "authcontract/v0.1.0"

    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.tree = Path(self.tmp.name) / "tree"
        copy_tree(self.tree)

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def contract(self, want_exit: int = 1) -> str:
        proc = run_gate(self.tree, "contract-check")
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, want_exit, f"contract-check exit {proc.returncode}, want {want_exit}:\n{out}")
        return out

    def assertReports(self, out: str, *wants: str) -> None:
        for want in wants:
            self.assertIn(want, out, f"contract-check did not report {want!r}:\n{out}")

    def write_schema(self, schema: dict) -> None:
        (self.tree / "authcontract/authcontract.schema.json").write_text(json.dumps(schema, indent=2) + "\n",
                                                                         encoding="utf-8")

    def test_dependency_outside_the_standard_library(self) -> None:
        """A package the module imports that is neither standard nor its own."""
        (self.tree / "authcontract/plant.go").write_text(
            '// Package authcontract is planted.\npackage authcontract\n\nimport _ "golang.org/x/term"\n', encoding="utf-8")
        self.assertReports(self.contract(), "authcontract depends on golang.org/x/term")

    def test_requirement_in_go_mod(self) -> None:
        """A requirement nothing imports: the module requires nothing."""
        with (self.tree / "authcontract/go.mod").open("a", encoding="utf-8") as f:
            f.write("\nrequire golang.org/x/sys v0.47.0\n")
        self.assertReports(self.contract(), "authcontract/go.mod requires golang.org/x/sys v0.47.0")

    def test_dependencies_admit_the_modules_own_package(self) -> None:
        """The module's own packages are admitted, and only by the module-prefix
        rule: with it removed, the tree's own package is named; restored, the
        tree's run passes."""
        script = self.tree / "scripts/check_contract.py"
        rule = 'return path == MODULE or path.startswith(MODULE + "/")'
        replace_once(script, rule, "return False")
        self.assertReports(self.contract(), "authcontract depends on github.com/maccavelli/go-llmprovider-sdk/authcontract,")
        replace_once(script, "return False", rule)
        self.assertReports(self.contract(0), "authcontract, 0 problem(s)")

    def test_api_removed_constant_against_the_tag(self) -> None:
        """An exported constant removed after the tag."""
        tag_tree(self.tree, self.TAG)
        replace_once(self.tree / "authcontract/step.go", 'StepWaiting StepKind = "waiting"\n', "")
        self.assertReports(self.contract(), "StepWaiting: removed", f"apidiff, against {self.TAG}")

    def test_api_type_change_at_v0(self) -> None:
        """A field's type changed after the tag: Go's rules allow it at v0, the
        gate does not (0073-MADR D7)."""
        tag_tree(self.tree, self.TAG)
        replace_once(self.tree / "authcontract/step.go", 'int                        `json:"seq"`',
                     'int64                      `json:"seq"`')
        self.assertReports(self.contract(), "Step.Seq", f"apidiff, against {self.TAG}: incompatible change")

    def test_schema_property_removed(self) -> None:
        """Step.URL's JSON name changed to uri, as a regenerated schema would
        show: apidiff does not report a tag, the schema comparison does."""
        self.write_schema(PLANT_SCHEMA)
        tag_tree(self.tree, self.TAG)
        replace_once(self.tree / "authcontract/step.go", '`json:"url,omitempty"`', '`json:"uri,omitempty"`')
        renamed = json.loads(json.dumps(PLANT_SCHEMA))
        props = renamed["$defs"]["Step"]["properties"]
        props["uri"] = props.pop("url")
        self.write_schema(renamed)
        out = self.contract()
        self.assertReports(out, f"schema, against {self.TAG}: property url removed from Step")
        self.assertNotIn("apidiff, against", out, f"apidiff reported a changed struct tag:\n{out}")

    def test_schema_enum_value_removed(self) -> None:
        """PasteCode deleted, and the enum value code with it."""
        self.write_schema(PLANT_SCHEMA)
        tag_tree(self.tree, self.TAG)
        replace_once(self.tree / "authcontract/method.go", '\t// PasteCode is a code only.\n\tPasteCode Paste = "code"\n', "")
        shorter = json.loads(json.dumps(PLANT_SCHEMA))
        shorter["$defs"]["Paste"]["enum"].remove("code")
        self.write_schema(shorter)
        self.assertReports(self.contract(), f"schema, against {self.TAG}: enum value code removed from Paste")

    SCHEMA = "authcontract/authcontract.schema.json"

    def regenerate(self) -> str:
        """Run the module's //go:generate line in the copy, as `go generate`
        does, and return the schema it wrote."""
        proc = subprocess.run(["go", "-C", "authcontract", "generate", "./..."], cwd=self.tree, capture_output=True,
                              text=True, check=False, env={**os.environ, "GOWORK": "off"})
        self.assertEqual(proc.returncode, 0, f"go generate exit {proc.returncode}:\n{proc.stdout}{proc.stderr}")
        return (self.tree / self.SCHEMA).read_text(encoding="utf-8")

    def tag_generated(self) -> None:
        """Tag the copy holding the schema the generator writes, unchanged by
        a run of it (0073-PLAN Phase 3)."""
        committed = (self.tree / self.SCHEMA).read_text(encoding="utf-8")
        self.assertEqual(self.regenerate(), committed, "the copy's schema is not what go generate writes")
        tag_tree(self.tree, self.TAG)

    def test_schema_property_removed_generated(self) -> None:
        """Plant 5 by the generator: Step.URL's JSON tag becomes uri after the
        tag, and go generate rewrites the schema. The schema comparison names
        url; apidiff, which does not read a tag, names nothing."""
        self.tag_generated()
        replace_once(self.tree / "authcontract/step.go", '`json:"url,omitempty"`', '`json:"uri,omitempty"`')
        props = json.loads(self.regenerate())["$defs"]["Step"]["properties"]
        self.assertIn("uri", props, "go generate did not write the renamed member")
        self.assertNotIn("url", props, "go generate kept the old member")
        out = self.contract()
        self.assertReports(out, f"schema, against {self.TAG}: property url removed from Step")
        self.assertNotIn("apidiff, against", out, f"apidiff reported a changed struct tag:\n{out}")

    def test_schema_enum_value_removed_generated(self) -> None:
        """Plant 6 by the generator: PasteCode deleted after the tag, and go
        generate rewrites the schema without the value code."""
        self.tag_generated()
        replace_once(self.tree / "authcontract/method.go", '\t// PasteCode is a code only.\n\tPasteCode Paste = "code"\n', "")
        enum = json.loads(self.regenerate())["$defs"]["Paste"]["enum"]
        self.assertNotIn("code", enum, "go generate kept the deleted constant's value")
        self.assertReports(self.contract(), f"schema, against {self.TAG}: enum value code removed from Paste")

    def test_schema_other_breaks(self) -> None:
        """The rest of D7's refusals: an entry removed, a type changed, a
        property newly required."""
        self.write_schema(PLANT_SCHEMA)
        tag_tree(self.tree, self.TAG)
        broken = json.loads(json.dumps(PLANT_SCHEMA))
        del broken["$defs"]["Paste"]
        broken["$defs"]["Step"]["properties"]["seq"]["type"] = "string"
        broken["$defs"]["Step"]["required"].append("url")
        self.write_schema(broken)
        self.assertReports(self.contract(), f"schema, against {self.TAG}: $defs entry Paste removed",
                           f"schema, against {self.TAG}: Step.seq: type changed from integer to string",
                           f"schema, against {self.TAG}: property url newly required in Step")

    def test_schema_additions_are_allowed(self) -> None:
        """Additions break nothing, end to end on the generated schema: after
        the tag, an enum constant and a new string type with one constant are
        added, go generate rewrites the schema, and contract-check passes. The
        additions no Go change can make while the module's own tests pass, an
        optional property and an object entry, are SchemaBreaksTest's
        (0073-PLAN Phase 3, the owner's decision of 2026-11-12)."""
        self.tag_generated()
        replace_once(self.tree / "authcontract/step.go", '\tStepFailed StepKind = "failed"\n',
                     '\tStepFailed StepKind = "failed"\n\t// StepPlanted is planted.\n\tStepPlanted StepKind = "planted"\n')
        (self.tree / "authcontract/planted.go").write_text(
            "package authcontract\n\n// Planted is planted. It is a value, safe for concurrent use while no\n"
            "// goroutine changes it.\ntype Planted string\n\n// PlantedOne is planted.\n"
            'const PlantedOne Planted = "one"\n', encoding="utf-8")
        defs = json.loads(self.regenerate())["$defs"]
        self.assertIn("planted", defs["StepKind"]["enum"], "go generate did not write the new constant")
        self.assertEqual(defs.get("Planted", {}).get("enum"), ["one"], "go generate did not write the new type")
        self.assertReports(self.contract(0), f"authcontract, against {self.TAG}, 0 problem(s)")

    def test_coverage_below_the_floor(self) -> None:
        """Every test file but one deleted: the package is under 80 %."""
        for test in (self.tree / "authcontract").glob("*_test.go"):
            if test.name != "example_test.go":
                test.unlink()
        self.assertReports(self.contract(), "authcontract is ", "%, below the 80.0% floor")

    def test_no_tag_passes_and_says_so(self) -> None:
        """Before the first authcontract/v* tag there is nothing to compare."""
        self.assertReports(self.contract(0), "no authcontract/v* tag yet, so nothing to compare")


class SchemaBreaksTest(unittest.TestCase):
    """check_contract.schema_breaks, called directly, for the additions no
    Go change can make while the module's own tests pass (0073-PLAN Phase 3,
    the owner's decision of 2026-11-12)."""

    def test_schema_breaks_ignore_additions(self) -> None:
        """A new entry, a new optional property and a new enum value, on
        PLANT_SCHEMA: no break."""
        sys.path.insert(0, str(ROOT / "scripts"))
        try:
            from check_contract import schema_breaks
        finally:
            sys.path.remove(str(ROOT / "scripts"))
        grown = json.loads(json.dumps(PLANT_SCHEMA))
        grown["$defs"]["Extra"] = {"type": "object", "properties": {}, "required": []}
        grown["$defs"]["Step"]["properties"]["more"] = {"type": "string"}
        grown["$defs"]["Paste"]["enum"].append("a_new_value")
        self.assertEqual(schema_breaks(PLANT_SCHEMA, grown), [], "an addition was reported as a break")


def load_check_changelog():
    """scripts/check_changelog.py, loaded by path as a module of its own."""
    spec = importlib.util.spec_from_file_location("check_changelog_0094", ROOT / "scripts/check_changelog.py")
    module = importlib.util.module_from_spec(spec)
    # A dataclass looks its module up by name while it is defined.
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class ChangelogReleasedTest(unittest.TestCase):
    """check_changelog.py's reader, called directly on the tree, which it
    only reads (0094-PLAN Phase 1)."""

    def setUp(self) -> None:
        self.cc = load_check_changelog()

    def test_missing_tag_commit_is_a_tool_error(self) -> None:
        """RS12 (b): a name whose commit the clone does not hold is a tool
        error, not a tag with no file."""
        with self.assertRaises(self.cc.ToolError) as caught:
            self.cc.read_files(["0" * 40], "CHANGELOG.md")
        self.assertIn("0" * 40, str(caught.exception))

    def test_git_calls_per_file(self) -> None:
        """RS16, as amended by 0094-MADR A1: per file, two `git cat-file`
        processes however many tags; one shallow check per run; no `git show`
        and no `git fetch`."""
        for namespace in ("v*", "authcontract/v*"):
            listed = subprocess.run(["git", "tag", "-l", namespace], cwd=ROOT, capture_output=True, text=True,
                                    check=True).stdout.split()
            self.assertTrue(listed, f"the tree has no {namespace} tag; this case needs both files' tags")
        calls: list[list[str]] = []
        real = subprocess.run

        def counting(args, *rest, **kwargs):
            calls.append(list(args))
            return real(args, *rest, **kwargs)

        out = io.StringIO()
        with mock.patch.object(self.cc.subprocess, "run", counting), contextlib.redirect_stdout(out):
            code = self.cc.main([])
        self.assertEqual(code, 0, out.getvalue())
        git = [c[1:] for c in calls if c[:1] == ["git"]]
        self.assertEqual(sum(1 for c in git if c[:1] == ["cat-file"]), 4, f"git calls: {git}")
        self.assertEqual(sum(1 for c in git if c == ["rev-parse", "--is-shallow-repository"]), 1, f"git calls: {git}")
        self.assertEqual([c for c in git if c[:1] in (["show"], ["fetch"])], [], f"git calls: {git}")


SELFTEST_STEP = re.compile(r"^\s*run: make .*\bgate-selftest\b", re.M)
GO_INSTALL = re.compile(r"\bgo install (\S+@\S+)")


def installs_before_selftest(workflow: Path) -> list[str] | None:
    """The `go install` targets a workflow runs before its `make …
    gate-selftest` line, in order; None when it has no such line."""
    text = workflow.read_text(encoding="utf-8")
    step = SELFTEST_STEP.search(text)
    if step is None:
        return None
    return GO_INSTALL.findall(text[:step.start()])


class WorkflowsTest(unittest.TestCase):
    """docs.yml runs gate-selftest as ci.yml does, so it installs each tool
    ci.yml installs first (0052-MADR A2): the precheck test needs
    golangci-lint, and the docs job without it failed every run."""

    def test_docs_installs_what_ci_installs_before_gate_selftest(self) -> None:
        workflows = ROOT / ".github/workflows"
        ci = installs_before_selftest(workflows / "ci.yml")
        docs = installs_before_selftest(workflows / "docs.yml")
        self.assertIsNotNone(ci, "ci.yml has no `make … gate-selftest` line")
        self.assertIsNotNone(docs, "docs.yml has no `make … gate-selftest` line")
        self.assertTrue(ci, "ci.yml installs nothing before gate-selftest; the precheck test needs golangci-lint")
        missing = [target for target in ci if target not in docs]
        self.assertEqual(missing, [], f"docs.yml does not install {missing} before gate-selftest, as ci.yml does; "
                                      f"it installs {docs}")


# The gates each workflow's `make … gate-selftest` line runs (0063-MADR,
# re-assessment). A Markdown gate reads Markdown, or plants in it, so a
# Markdown-only change, which only docs.yml runs for, must run it; a code
# gate reads no Markdown and runs in ci.yml alone. A gate in neither tuple
# fails the contract until it is placed in one.
MARKDOWN_GATES = ("parity-check", "records-check", "tables-check", "changelog-check", "markdownlint", "links-check",
                  "disclosure-check", "gate-selftest")
CODE_GATES = ("dep-check", "coverage-check", "api-check", "contract-check", "generate-check", "fixtures-check",
              "docs-check")
EVENT_LINE = re.compile(r"^  ([a-z_]+):\s*$")
KEY_LINE = re.compile(r"^    ([a-z-]+):\s*\[(.*)\]\s*$")
ITEM_LINE = re.compile(r"^    - (.+)$")
USES = re.compile(r"^\s*-?\s*uses: (\S+)", re.M)


def triggers(workflow: Path) -> dict[str, dict[str, list[str]]]:
    """A workflow's `on:` block: each event, its filters' flow lists, and
    its list items under "-". A line of any other shape fails, so a filter
    written another way is not passed over (0063-MADR R4)."""
    lines = workflow.read_text(encoding="utf-8").splitlines()
    out: dict[str, dict[str, list[str]]] = {}
    event = None
    for line in lines[lines.index("on:") + 1:]:
        if line and not line.startswith((" ", "#")):
            break
        if not line.strip() or line.strip().startswith("#"):
            continue
        if m := EVENT_LINE.match(line):
            event = m.group(1)
            out[event] = {}
        elif event and (m := KEY_LINE.match(line)):
            out[event][m.group(1)] = [item.strip().strip("'\"") for item in m.group(2).split(",") if item.strip()]
        elif event and (m := ITEM_LINE.match(line)):
            out[event].setdefault("-", []).append(m.group(1))
        else:
            raise AssertionError(f"{workflow.name}: an `on:` line of a shape the contract does not read: {line!r}")
    return out


def selftest_targets(workflow: Path) -> list[str]:
    """The targets of a workflow's `make … gate-selftest` line."""
    step = SELFTEST_STEP.search(workflow.read_text(encoding="utf-8"))
    if step is None:
        raise AssertionError(f"{workflow.name} has no `make … gate-selftest` line")
    return step.group(0).split("make", 1)[1].split()


class WorkflowContractTest(unittest.TestCase):
    """The routing of 0052-MADR and the shared gates of 0063-MADR: what
    ci.yml skips, docs.yml runs, with the same gates and pins (0063-PLAN
    Phase 2, step 2)."""

    def setUp(self) -> None:
        self.ci = ROOT / ".github/workflows/ci.yml"
        self.docs = ROOT / ".github/workflows/docs.yml"

    def test_ci_keeps_its_triggers_and_gains_no_paths_filter(self) -> None:
        """Every tag, each push to main, each pull request and the weekly run
        reach ci.yml; only paths-ignore narrows it, the same for push and
        pull request."""
        on = triggers(self.ci)
        for event in ("push", "pull_request", "schedule"):
            self.assertIn(event, on, f"ci.yml lost its {event} trigger")
        self.assertIn("v*", on["push"].get("tags", []), "ci.yml no longer runs for a v* tag")
        self.assertIn("main", on["push"].get("branches", []), "ci.yml no longer runs for a push to main")
        self.assertTrue(on["schedule"].get("-"), "ci.yml's schedule has no cron")
        for event in ("push", "pull_request"):
            self.assertNotIn("paths", on[event], f"ci.yml's {event} gained a paths filter")
        self.assertEqual(on["push"].get("paths-ignore"), on["pull_request"].get("paths-ignore"),
                         "ci.yml ignores different paths for push and pull request")

    def test_docs_runs_for_every_path_ci_ignores(self) -> None:
        """A change ci.yml skips starts docs.yml, and so does a change to
        docs.yml itself."""
        ignored = triggers(self.ci)["push"].get("paths-ignore", [])
        on = triggers(self.docs)
        self.assertEqual(on["push"].get("paths"), on["pull_request"].get("paths"),
                         "docs.yml runs for different paths on push and pull request")
        paths = on["push"].get("paths", [])
        self.assertEqual([p for p in ignored if p not in paths], [], f"docs.yml does not run for {ignored} ci.yml skips")
        self.assertIn(".github/workflows/docs.yml", paths, "docs.yml does not run for a change to itself")
        for event in ("push", "pull_request"):
            self.assertNotIn("paths-ignore", on[event], f"docs.yml's {event} gained a paths-ignore filter")

    def test_each_gate_is_placed_and_the_markdown_gates_run_in_both(self) -> None:
        """Every gate ci.yml runs is a Markdown or a code gate; docs.yml runs
        exactly the Markdown gates, and ci.yml runs them and every code gate
        (0073-MADR D8: a code gate dropped from ci.yml was passed unseen)."""
        ci, docs = selftest_targets(self.ci), selftest_targets(self.docs)
        unplaced = [t for t in ci + docs if t not in MARKDOWN_GATES + CODE_GATES]
        self.assertEqual(unplaced, [], f"gates neither MARKDOWN_GATES nor CODE_GATES: {unplaced}")
        self.assertEqual([g for g in MARKDOWN_GATES if g not in docs], [],
                         "docs.yml does not run these Markdown gates, so a Markdown-only change skips them")
        self.assertEqual([g for g in MARKDOWN_GATES if g not in ci], [], "ci.yml does not run these Markdown gates")
        self.assertEqual([g for g in CODE_GATES if g not in ci], [], "ci.yml does not run these code gates")
        self.assertEqual([g for g in docs if g in CODE_GATES], [], "docs.yml runs code gates, which read no Markdown")

    def test_docs_uses_ci_actions_and_pins(self) -> None:
        """docs.yml's actions are ci.yml's, at the same commits, and neither
        sets markdownlint's pin, which the Makefile holds alone."""
        ci_uses = set(USES.findall(self.ci.read_text(encoding="utf-8")))
        docs_uses = USES.findall(self.docs.read_text(encoding="utf-8"))
        self.assertTrue(docs_uses, "docs.yml uses no action")
        self.assertEqual([u for u in docs_uses if u not in ci_uses], [],
                         f"docs.yml uses actions or commits ci.yml does not: {sorted(ci_uses)}")
        for workflow in (self.ci, self.docs):
            self.assertNotIn("MARKDOWNLINT", workflow.read_text(encoding="utf-8"),
                             f"{workflow.name} sets markdownlint's pin; the Makefile holds it")


if __name__ == "__main__":
    unittest.main()
