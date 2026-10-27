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

import os
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
    "tables-check": "scripts/check_tables.py",
    "fixtures-check": "scripts/check_fixtures.py",
    "changelog-check": "scripts/check_changelog.py",
    "docs-check": "scripts/check_docs.py",
}
GENERATE_LINE = re.compile(r"^//go:generate .*$", re.M)


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
        (0046-MADR D4)."""
        replace_once(self.tree / "docs/architecture.md", "`providers/openai`, `internal/redact`, `golang.org/x/term` |",
                     "`providers/openai`, `internal/redact`, `internal/transport`, `golang.org/x/term` |")
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

    def test_clean_copy_passes_every_gate(self) -> None:
        for gate in GATES:
            with self.subTest(gate=gate):
                proc = run_gate(self.tree, gate)
                self.assertEqual(proc.returncode, 0, f"{gate} exit {proc.returncode} on a clean copy:\n"
                                                     f"{proc.stdout}{proc.stderr}")


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


if __name__ == "__main__":
    unittest.main()
