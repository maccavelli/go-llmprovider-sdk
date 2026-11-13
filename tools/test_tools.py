#!/usr/bin/env python3
"""tools-test: each maintainer tool fails on a planted input, and passes a clean one.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D3, and
written in 0035-PLAN Phase 1. Every case runs in a temporary directory and
never touches the tree. An identifier the identifier scan looks for is built
at run time, so this file holds none.

Usage:
  make tools-test
  python3 -B -m unittest discover -s tools -p 'test_*.py'
"""
from __future__ import annotations

import contextlib
import getpass
import io
import json
import os
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import _common
import apidiff
import cachecensus
import catalogcheck
import claims
import consumer
import denyscan
import edit
import gate
import leakfind
import linkcheck
import liveskips
import offline
import plant
import recordlint
import records
import redgreen


def git(cwd: Path, *args: str) -> None:
    subprocess.run(["git", *args], cwd=cwd, check=True, capture_output=True)


def temp_repo(test: unittest.TestCase) -> Path:
    """A fresh git repository in a temporary directory, removed after the test."""
    tmp = tempfile.TemporaryDirectory()
    test.addCleanup(tmp.cleanup)
    root = Path(tmp.name)
    git(root, "init", "-q")
    git(root, "config", "user.email", "tools-test@invalid")
    git(root, "config", "user.name", "tools-test")
    return root


def call(main, *argv: str) -> tuple[int, str]:
    """Run a tool's main with argv, returning its exit code and its output."""
    out = io.StringIO()
    with contextlib.redirect_stdout(out), contextlib.redirect_stderr(out):
        rc = main(list(argv))
    return rc, out.getvalue()


class TestRoot(unittest.TestCase):
    def test_root_holds_go_mod(self) -> None:
        self.assertTrue((_common.ROOT / "go.mod").is_file())

    def test_root_does_not_follow_the_working_directory(self) -> None:
        before = os.getcwd()
        self.addCleanup(os.chdir, before)
        with tempfile.TemporaryDirectory() as tmp:
            os.chdir(tmp)
            self.assertEqual(_common.ROOT, Path(_common.__file__).resolve().parent.parent)


class TestWaitLoad(unittest.TestCase):
    def test_waits_until_the_load_falls(self) -> None:
        loads, slept = iter([50.0, 1.0]), []
        ok = _common.wait_load(10, 60, load=lambda: next(loads), sleep=slept.append, clock=lambda: 0.0)
        self.assertTrue(ok)
        self.assertEqual(len(slept), 1)

    def test_gives_up_at_the_timeout(self) -> None:
        ok = _common.wait_load(10, 0, load=lambda: 50.0, sleep=lambda _: None, clock=lambda: 0.0)
        self.assertFalse(ok)


class TestLinkCheck(unittest.TestCase):
    def test_a_missing_target_is_a_finding(self) -> None:
        root = temp_repo(self)
        (root / "a.md").write_text("ok [b](b.md)\nbad [gone](missing.md)\n")
        (root / "b.md").write_text("back [a](a.md#top)\n")
        rc, out = call(linkcheck.main, "--root", str(root))
        self.assertEqual(rc, 1, out)
        self.assertIn("a.md:2: missing.md", out)
        self.assertNotIn("b.md", out.split("G-links:")[0])

    def test_fences_urls_and_anchors_are_not_checked(self) -> None:
        root = temp_repo(self)
        (root / "a.md").write_text("```\n[x](nowhere.md)\n```\n[w](https://example.com) [h](#here) `[c](code.md)`\n")
        rc, out = call(linkcheck.main, "--root", str(root))
        self.assertEqual(rc, 0, out)


class TestDenyScan(unittest.TestCase):
    def scan(self, text: str) -> tuple[int, str]:
        root = temp_repo(self)
        (root / "f.md").write_text(text)
        return call(denyscan.main, "--root", str(root), "f.md")

    def test_the_account_name_is_a_hit_and_is_never_printed(self) -> None:
        name = getpass.getuser()
        if len(name) < 3:
            self.skipTest("an account name under three characters is not scanned for")
        rc, out = self.scan(f"written by {name} today\n")
        self.assertEqual(rc, 1, out)
        self.assertIn("f.md:1: the local account name", out)
        self.assertNotIn(name, out)

    def test_a_placeholder_home_path_is_not_a_hit(self) -> None:
        rc, out = self.scan("see /home/<user>/x and /Users/<user>/y\n")
        self.assertEqual(rc, 0, out)

    def test_a_named_home_path_is_a_hit(self) -> None:
        path = "/" + "Users" + "/" + "someone" + "/repo"
        rc, out = self.scan(f"clone at {path}\n")
        self.assertEqual(rc, 1, out)
        self.assertIn("f.md:1: a machine home path", out)

    def test_a_windows_home_path_is_a_hit(self) -> None:
        """disclosure-check's Windows form, which this tool imports (0063-MADR R3)."""
        path = "C:" + "\\" + "Users" + "\\" + "someone" + "\\repo"
        rc, out = self.scan(f"clone at {path}\n")
        self.assertEqual(rc, 1, out)
        self.assertIn("f.md:1: a machine home path", out)
        self.assertNotIn("someone", out)

    def test_changed_scans_the_files_git_lists(self) -> None:
        root = temp_repo(self)
        (root / "clean.md").write_text("nothing here\n")
        git(root, "add", "clean.md")
        git(root, "commit", "-q", "-m", "seed")
        (root / "new.md").write_text("clone at /" + "home" + "/someone/repo\n")
        rc, out = call(denyscan.main, "--root", str(root), "--changed")
        self.assertEqual(rc, 1, out)
        self.assertIn("new.md:1: a machine home path", out)
        self.assertIn("in 1 file(s)", out)


class TestEdit(unittest.TestCase):
    def setUp(self) -> None:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.root = Path(tmp.name)
        (self.root / "a.txt").write_text("one two two\n")
        (self.root / "b.txt").write_text("alpha\n")

    def apply(self, spec: list[dict]) -> tuple[int, str]:
        path = self.root / "spec.json"
        path.write_text(json.dumps(spec))
        return call(edit.main, "--root", str(self.root), str(path))

    def test_a_single_match_is_applied(self) -> None:
        rc, out = self.apply([{"file": "a.txt", "old": "one", "new": "1"}, {"file": "b.txt", "old": "alpha", "new": "beta"}])
        self.assertEqual(rc, 0, out)
        self.assertEqual((self.root / "a.txt").read_text(), "1 two two\n")
        self.assertEqual((self.root / "b.txt").read_text(), "beta\n")

    def test_a_double_match_changes_nothing(self) -> None:
        rc, out = self.apply([{"file": "b.txt", "old": "alpha", "new": "beta"}, {"file": "a.txt", "old": "two", "new": "2"}])
        self.assertEqual(rc, 1, out)
        self.assertIn("a.txt: 'two' matches 2 time(s), want 1", out)
        self.assertEqual((self.root / "a.txt").read_text(), "one two two\n")
        self.assertEqual((self.root / "b.txt").read_text(), "alpha\n")

    def test_no_match_changes_nothing(self) -> None:
        rc, out = self.apply([{"file": "a.txt", "old": "three", "new": "3"}])
        self.assertEqual(rc, 1, out)
        self.assertEqual((self.root / "a.txt").read_text(), "one two two\n")

    def test_a_declared_count_is_honoured(self) -> None:
        rc, out = self.apply([{"file": "a.txt", "old": "two", "new": "2", "count": 2}])
        self.assertEqual(rc, 0, out)
        self.assertEqual((self.root / "a.txt").read_text(), "one 2 2\n")

    def test_an_append_adds_at_the_end(self) -> None:
        (self.root / "c.txt").write_text("no final newline")
        rc, out = self.apply([{"file": "a.txt", "append": "three\n"}, {"file": "c.txt", "append": "next\n"}])
        self.assertEqual(rc, 0, out)
        self.assertEqual((self.root / "a.txt").read_text(), "one two two\nthree\n")
        self.assertEqual((self.root / "c.txt").read_text(), "no final newline\nnext\n")

    def test_an_append_follows_an_earlier_replace(self) -> None:
        rc, out = self.apply([{"file": "b.txt", "old": "alpha", "new": "beta"}, {"file": "b.txt", "append": "gamma\n"}])
        self.assertEqual(rc, 0, out)
        self.assertEqual((self.root / "b.txt").read_text(), "beta\ngamma\n")

    def test_a_create_makes_a_new_file(self) -> None:
        rc, out = self.apply([{"file": "new/d.txt", "create": "delta\n"}])
        self.assertEqual(rc, 0, out)
        self.assertEqual((self.root / "new/d.txt").read_text(), "delta\n")

    def test_a_create_of_an_existing_file_changes_nothing(self) -> None:
        rc, out = self.apply([{"file": "b.txt", "old": "alpha", "new": "beta"}, {"file": "a.txt", "create": "x\n"}])
        self.assertEqual(rc, 1, out)
        self.assertIn("a.txt: exists; create wants a new file", out)
        self.assertEqual((self.root / "a.txt").read_text(), "one two two\n")
        self.assertEqual((self.root / "b.txt").read_text(), "alpha\n")

    def test_text_from_files_beside_the_spec(self) -> None:
        (self.root / "old.txt").write_text("two two\n")
        (self.root / "new.txt").write_text("2\n2\n")
        rc, out = self.apply([{"file": "a.txt", "old_file": "old.txt", "new_file": "new.txt"}])
        self.assertEqual(rc, 0, out)
        self.assertEqual((self.root / "a.txt").read_text(), "one 2\n2\n")

    def test_a_missing_text_file_is_a_tool_error(self) -> None:
        rc, out = self.apply([{"file": "a.txt", "old_file": "absent.txt", "new": "x"}])
        self.assertEqual(rc, 2, out)
        self.assertEqual((self.root / "a.txt").read_text(), "one two two\n")

    def test_an_entry_of_two_shapes_is_a_tool_error(self) -> None:
        rc, out = self.apply([{"file": "a.txt", "old": "one", "new": "1", "append": "x"}])
        self.assertEqual(rc, 2, out)
        rc, out = self.apply([{"file": "a.txt", "old": "one", "old_file": "x", "new": "1"}])
        self.assertEqual(rc, 2, out)


INDEX = """# Documentation

## Records

2 records. Cite them by full filename.

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | MADR | [First](decisions/0001-MADR-first.md) | accepted |
| 0003 | MADR | [Third](decisions/0003-MADR-third.md) | proposed |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| know the first | [0001-MADR](decisions/0001-MADR-first.md) |
"""


class TestRecords(unittest.TestCase):
    def setUp(self) -> None:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.root = Path(tmp.name)
        (self.root / "docs/decisions").mkdir(parents=True)
        (self.root / "docs/reports").mkdir()
        (self.root / "docs/README.md").write_text(INDEX)
        self.write("0001-MADR-first.md", "accepted", "First")
        self.write("0003-MADR-third.md", "proposed", "Third")

    def write(self, name: str, status: str, title: str, extra: str = "") -> Path:
        path = self.root / "docs/decisions" / name
        path.write_text(f"---\nstatus: {status}\ndate: 2026-10-01\n{extra}---\n\n# {title}\n\nBody.\n")
        return path

    def run_records(self, *argv: str) -> tuple[int, str]:
        return call(records.main, "--root", str(self.root), *argv)

    def test_add_places_a_row_by_number_and_recounts(self) -> None:
        self.write("0002-MADR-second.md", "proposed", "Second")
        rc, out = self.run_records("add", "docs/decisions/0002-MADR-second.md")
        self.assertEqual(rc, 0, out)
        text = (self.root / "docs/README.md").read_text()
        rows = [line for line in text.splitlines() if line.startswith("| 000")]
        self.assertEqual([r[:13] for r in rows], ["| 0001 | MADR", "| 0002 | MADR", "| 0003 | MADR"])
        self.assertIn("| 0002 | MADR | [Second](decisions/0002-MADR-second.md) | proposed |", text)
        self.assertIn("3 records. Cite them by full filename.", text)
        self.assertEqual(text.split("## I want to…")[1], INDEX.split("## I want to…")[1])

    def test_add_a_plan_after_its_madr(self) -> None:
        self.write("0003-PLAN-third.md", "proposed", "Implement Third", 'associated-madr: "0003-MADR-third.md"\n')
        rc, out = self.run_records("add", "docs/decisions/0003-PLAN-third.md")
        self.assertEqual(rc, 0, out)
        rows = [l for l in (self.root / "docs/README.md").read_text().splitlines() if l.startswith("| 000")]
        self.assertEqual(rows[-1], "| 0003 | PLAN | [Implement Third](decisions/0003-PLAN-third.md) | proposed |")

    def test_add_of_an_indexed_record_writes_nothing(self) -> None:
        rc, out = self.run_records("add", "docs/decisions/0001-MADR-first.md")
        self.assertEqual(rc, 1, out)
        self.assertIn("already indexed", out)
        self.assertEqual((self.root / "docs/README.md").read_text(), INDEX)

    def test_status_sets_the_front_matter_and_the_row(self) -> None:
        rc, out = self.run_records("status", "docs/decisions/0003-MADR-third.md", "accepted", "--date", "2026-10-19")
        self.assertEqual(rc, 0, out)
        record = (self.root / "docs/decisions/0003-MADR-third.md").read_text()
        self.assertTrue(record.startswith("---\nstatus: accepted\ndate: 2026-10-19\n---\n"), record)
        self.assertIn("| 0003 | MADR | [Third](decisions/0003-MADR-third.md) | accepted |",
                      (self.root / "docs/README.md").read_text())

    def test_a_bad_status_writes_nothing(self) -> None:
        before = (self.root / "docs/decisions/0003-MADR-third.md").read_text()
        rc, out = self.run_records("status", "docs/decisions/0003-MADR-third.md", "complete")
        self.assertEqual(rc, 2, out)
        self.assertEqual((self.root / "docs/decisions/0003-MADR-third.md").read_text(), before)
        self.assertEqual((self.root / "docs/README.md").read_text(), INDEX)

    def test_the_checks_run_after_writing(self) -> None:
        self.write("0002-PLAN-second.md", "proposed", "Implement Second", 'associated-madr: "0002-MADR-second.md"\n')
        rc, out = self.run_records("add", "docs/decisions/0002-PLAN-second.md")
        self.assertEqual(rc, 1, out)
        self.assertIn("no MADR numbered 0002", out)

    def test_open_lists_the_open_records_in_number_order(self) -> None:
        self.write("0001-PLAN-first.md", "complete", "Implement First", 'associated-madr: "0001-MADR-first.md"\n')
        self.write("0003-PLAN-third.md", "in-progress", "Implement Third", 'associated-madr: "0003-MADR-third.md"\n')
        (self.root / "docs/reports/0002-REPORT-second.md").write_text("---\nstatus: observation\n---\n\n# Second\n")
        before = {p: p.read_text() for p in self.root.rglob("*.md")}
        rc, out = self.run_records("open")
        self.assertEqual(rc, 0, out)
        self.assertEqual(out.splitlines(), [
            "proposed     2026-10-01 docs/decisions/0003-MADR-third.md",
            "in-progress  2026-10-01 docs/decisions/0003-PLAN-third.md",
            "records: 2 open",
        ])
        self.assertEqual({p: p.read_text() for p in self.root.rglob("*.md")}, before)

    def test_the_open_statuses_are_check_records(self) -> None:
        self.assertEqual(set(records.OPEN), {"MADR", "PLAN"})
        self.assertLessEqual(set(records.OPEN["MADR"]), set(records.check_records.MADR_STATUSES))
        self.assertLessEqual(set(records.OPEN["PLAN"]), set(records.check_records.PLAN_STATUSES))


RECORD = "docs/decisions/0001-MADR-lint.md"
RECORD_TEXT = "# Lint\n\n* one\n* two\n\nProse.\n"


@unittest.skipUnless(shutil.which("markdownlint-cli2"), "markdownlint-cli2 is not on PATH")
class TestRecordLint(unittest.TestCase):
    """The repository config pins "-" markers; the records' "*" must not count (A2)."""

    def setUp(self) -> None:
        self.root = temp_repo(self)
        (self.root / ".markdownlint-cli2.jsonc").write_text(
            '{\n  // the repository\'s rules\n  "config": {"MD004": {"style": "dash"}, "MD013": false,},\n}\n')
        (self.root / "docs/decisions").mkdir(parents=True)
        (self.root / RECORD).write_text(RECORD_TEXT)
        git(self.root, "add", ".")
        git(self.root, "commit", "-q", "-m", "seed")

    def lint(self, *argv: str) -> tuple[int, str]:
        return call(recordlint.main, "--root", str(self.root), *argv)

    def test_an_added_finding_is_named_with_its_line(self) -> None:
        (self.root / RECORD).write_text(RECORD_TEXT + "\n\n\nMore.\n")
        rc, out = self.lint(RECORD)
        self.assertEqual(rc, 1, out)
        self.assertIn(f"{RECORD}: new findings", out)
        self.assertRegex(out, rf"{RECORD}:\d+ MD012")

    def test_old_findings_alone_pass(self) -> None:
        (self.root / RECORD).write_text(RECORD_TEXT + "\n\n\nOld.\n")
        git(self.root, "commit", "-q", "-am", "an old finding")
        (self.root / RECORD).write_text(RECORD_TEXT + "\n\n\nOld.\n\nNew prose.\n")
        rc, out = self.lint(RECORD)
        self.assertEqual(rc, 0, out)
        self.assertIn("HEAD {'MD012': 2}, tree {'MD012': 2}", out)

    def test_star_markers_and_tabs_in_code_are_the_records_convention(self) -> None:
        (self.root / RECORD).write_text(RECORD_TEXT + "\n* three\n\n```go\nfunc f() {\n\treturn\n}\n```\n")
        rc, out = self.lint(RECORD)
        self.assertEqual(rc, 0, out)

    def test_mixed_markers_are_a_finding(self) -> None:
        (self.root / RECORD).write_text(RECORD_TEXT + "\n- three\n")
        rc, out = self.lint(RECORD)
        self.assertEqual(rc, 1, out)
        self.assertRegex(out, rf"{RECORD}:\d+ MD004")

    def test_a_new_record_is_compared_with_none(self) -> None:
        new = "docs/decisions/0002-MADR-new.md"
        (self.root / new).write_text("# New\n\n\n\nText.\n")
        rc, out = self.lint("--changed")
        self.assertEqual(rc, 1, out)
        self.assertIn(f"{new}: new findings; HEAD {{}}", out)
        self.assertNotIn(f"{RECORD}:", out)


class TestScratchClone(unittest.TestCase):
    def test_the_clone_commits_the_working_tree(self) -> None:
        root = temp_repo(self)
        for name in ("kept.txt", "changed.txt", "deleted.txt"):
            (root / name).write_text(name + "\n")
        git(root, "add", ".")
        git(root, "commit", "-q", "-m", "seed")
        (root / "changed.txt").write_text("changed\n")
        (root / "new.txt").write_text("new\n")
        git(root, "rm", "-q", "deleted.txt")
        with tempfile.TemporaryDirectory() as tmp:
            dest = Path(tmp) / "clone"
            changed = _common.scratch_clone(root, dest)
            tree = subprocess.run(["git", "ls-tree", "-r", "--name-only", "HEAD"], cwd=dest,
                                  capture_output=True, text=True, check=True).stdout.split()
            self.assertEqual(sorted(tree), ["changed.txt", "kept.txt", "new.txt"])
            self.assertEqual((dest / "changed.txt").read_text(), "changed\n")
            self.assertEqual(sorted(changed), ["changed.txt", "new.txt"])


class TestGateChecks(unittest.TestCase):
    def test_the_checks_and_their_order(self) -> None:
        want = ["pre-add-check", "vet-darwin", "vet-darwin-live", "vet-linux", "vet-linux-live", "vet-windows",
                "vet-windows-live", "test-race-cover", "test-shuffle", "test-repeat", "tidy-diff", "lint", "parity-check",
                "dep-check",
                "coverage-check", "api-check", "contract-check", "generate-check", "records-check", "tables-check", "fixtures-check",
                "changelog-check", "docs-check", "gate-selftest", "markdownlint",
                "g-wire-stable", "links-check", "disclosure-check", "deny-scan", "tools-test"]
        self.assertEqual([c.name for c in gate.CHECKS], want)

    def test_the_markdown_checks_are_ci_docs_gates_then_the_scan_lint_and_claims(self) -> None:
        names = [c.name for c in gate.markdown_checks(_common.ROOT)]
        self.assertEqual(names, [*gate.MARKDOWN_GATES, "deny-scan", "recordlint", "claims"])
        self.assertIn("gate-selftest", names)

    def test_markdown_refuses_a_change_that_is_not_markdown(self) -> None:
        root = temp_repo(self)
        for name in ("notes.md", "code.go", "gone.go"):
            (root / name).write_text("x\n")
        git(root, "add", ".")
        git(root, "commit", "-q", "-m", "seed")
        (root / "notes.md").write_text("y\n")
        (root / "code.go").write_text("y\n")
        (root / "gone.go").unlink()
        work = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, work, ignore_errors=True)
        with mock.patch.object(gate, "wait_load", return_value=True):
            rc, out = call(gate.main, "--markdown", "--src", str(root), "--copy", str(work / "copy"), "--out",
                           str(work / "out"))
        self.assertEqual(rc, 1, out)
        self.assertIn("gate: code.go is not Markdown", out)
        self.assertIn("gate: gone.go is not Markdown", out)
        self.assertNotIn("notes.md is not", out)
        self.assertIn("gate --markdown: nothing run", out)
        self.assertFalse((work / "out" / "summary.out").exists())

    def test_markdown_claims_take_the_changed_records_with_an_index_and_none_is_clean(self) -> None:
        src = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, src, ignore_errors=True)
        (src / "tools").mkdir()
        (src / "tools/claims.py").write_text("import sys\nprint('given', sys.argv[1:])\n")
        (src / "docs/decisions").mkdir(parents=True)
        (src / "docs/reports").mkdir()
        indexed = ["docs/decisions/0001-PLAN-a-slug.md", "docs/reports/0002-REPORT-b.md"]
        for rel in indexed:
            (src / rel).write_text("# A\n\n<!-- claims: self=abc1234 -->\n")
        (src / "docs/decisions/0003-MADR-c.md").write_text("# C\n\nNo index; `<!-- claims: x=y -->` quoted.\n")
        (src / "docs/decisions/notes.md").write_text("<!-- claims: self=abc1234 -->\n")
        none = "claims: no record with an evidence index changed\n"
        check = next(c for c in gate.markdown_checks(src) if c.name == "claims")
        self.assertEqual(check.run(src, ["AGENTS.md", "docs/architecture.md"]), (0, none))
        self.assertEqual(check.run(src, ["docs/decisions/0003-MADR-c.md"]), (0, none))
        rc, out = check.run(src, ["AGENTS.md", indexed[0], "docs/decisions/0003-MADR-c.md", "docs/decisions/notes.md",
                                  indexed[1]])
        self.assertEqual(rc, 0, out)
        self.assertEqual(out.strip(), f"given {indexed!r}")

    def test_markdown_waits_for_the_load(self) -> None:
        root = temp_repo(self)
        (root / "notes.md").write_text("x\n")
        git(root, "add", ".")
        git(root, "commit", "-q", "-m", "seed")
        (root / "notes.md").write_text("y\n")
        work = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, work, ignore_errors=True)
        with mock.patch.object(gate, "wait_load", return_value=False) as waited:
            rc, out = call(gate.main, "--markdown", "--src", str(root), "--copy", str(work / "copy"), "--out",
                           str(work / "out"))
        self.assertEqual(rc, 2, out)
        waited.assert_called_once()
        self.assertIn("nothing run", out)
        self.assertFalse((work / "copy").exists())


GO_MOD = "module example.com/plant\n\ngo 1.21\n"
GO_SRC = "package plant\n\n// Add returns the sum of its arguments.\nfunc Add(a, b int) int { return a + b }\n"
GO_TEST = ("package plant\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n"
           "\tif got := Add(1, 2); got != 3 {\n\t\tt.Errorf(\"Add(1, 2) = %d, want 3\", got)\n\t}\n}\n")


class TestPlant(unittest.TestCase):
    def run_spec(self, plants: list[dict]) -> tuple[int, str]:
        root = temp_repo(self)
        (root / "go.mod").write_text(GO_MOD)
        (root / "plant.go").write_text(GO_SRC)
        (root / "plant_test.go").write_text(GO_TEST)
        git(root, "add", ".")
        git(root, "commit", "-q", "-m", "seed")
        spec = root.parent / (root.name + "-spec.json")
        self.addCleanup(spec.unlink, missing_ok=True)
        spec.write_text(json.dumps({"packages": ["./..."], "run": "TestAdd", "plants": plants}))
        return call(plant.main, "--root", str(root), str(spec))

    def test_a_caught_plant_passes_with_its_fail_lines(self) -> None:
        rc, out = self.run_spec([{"name": "subtracts", "file": "plant.go", "old": "a + b", "new": "a - b"}])
        self.assertEqual(rc, 0, out)
        self.assertIn("== subtracts: caught", out)
        self.assertIn("Add(1, 2) = -1, want 3", out)

    def test_an_uncaught_plant_fails_naming_it(self) -> None:
        rc, out = self.run_spec([{"name": "commutes", "file": "plant.go", "old": "a + b", "new": "b + a"}])
        self.assertEqual(rc, 1, out)
        self.assertIn("== commutes: NOT CAUGHT", out)

    def test_a_plant_from_files_is_caught_as_its_inline_twin(self) -> None:
        spec_dir = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, spec_dir, ignore_errors=True)
        (spec_dir / "old.txt").write_text("a + b")
        (spec_dir / "new.txt").write_text("a - b")
        rc, out = self.run_spec([{"name": "subtracts", "file": "plant.go", "old_file": str(spec_dir / "old.txt"),
                                  "new_file": str(spec_dir / "new.txt")}])
        self.assertEqual(rc, 0, out)
        self.assertIn("== subtracts: caught", out)
        self.assertIn("Add(1, 2) = -1, want 3", out)

    def test_a_plant_that_does_not_match_is_a_tool_error(self) -> None:
        rc, out = self.run_spec([{"name": "absent", "file": "plant.go", "old": "a * b", "new": "a / b"}])
        self.assertEqual(rc, 2, out)
        self.assertIn("absent", out)


RG_MOD = "module example.com/rg\n\ngo 1.21\n"
RG_BUGGY = "package rg\n\n// Add returns the sum of its arguments.\nfunc Add(a, b int) int { return a - b }\n"
RG_FIXED = ("package rg\n\n// Add returns the sum of its arguments.\nfunc Add(a, b int) int { return a + b }\n\n"
            "// Sub returns a less b.\nfunc Sub(a, b int) int { return a - b }\n")
RG_ZERO_TEST = ("package rg\n\nimport \"testing\"\n\nfunc TestZero(t *testing.T) {\n"
                "\tif Add(0, 0) != 0 {\n\t\tt.Error(\"Add(0, 0) != 0\")\n\t}\n}\n")
RG_FIX_TEST = ("package rg\n\nimport \"testing\"\n\n// TestAddSums is red before the fix.\n"
               "func TestAddSums(t *testing.T) {\n\tif got := Add(1, 2); got != 3 {\n"
               "\t\tt.Errorf(\"Add(1, 2) = %d, want 3\", got)\n\t}\n}\n\n"
               "// TestSub needs Sub, which the first commit lacks.\n// A second comment line.\n"
               "func TestSub(t *testing.T) {\n\tif Sub(3, 1) != 2 {\n\t\tt.Error(\"Sub(3, 1) != 2\")\n\t}\n}\n")


@unittest.skipUnless(shutil.which("go"), "go is not on PATH")
class TestRedGreen(unittest.TestCase):
    """0067-MADR D1: a module whose first commit has a bug that its second fixes, with the fix's tests."""

    def setUp(self) -> None:
        self.root = temp_repo(self)
        (self.root / "go.mod").write_text(RG_MOD)
        (self.root / "rg.go").write_text(RG_BUGGY)
        (self.root / "zero_test.go").write_text(RG_ZERO_TEST)
        git(self.root, "add", ".")
        git(self.root, "commit", "-q", "-m", "buggy")
        self.first = subprocess.run(["git", "rev-parse", "HEAD"], cwd=self.root, capture_output=True, text=True,
                                    check=True).stdout.strip()
        (self.root / "rg.go").write_text(RG_FIXED)
        (self.root / "fix_test.go").write_text(RG_FIX_TEST)
        git(self.root, "add", ".")
        git(self.root, "commit", "-q", "-m", "fixed")
        self.out = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, self.out, ignore_errors=True)

    def run_spec(self, spec: dict, *args: str) -> tuple[int, str]:
        path = self.out / "spec.json"
        path.write_text(json.dumps(spec))
        return call(redgreen.main, "--root", str(self.root), "--out", str(self.out), str(path), *args)

    def test_green_on_the_tree(self) -> None:
        rc, out = self.run_spec({"tests": {".": ["TestZero", "TestAddSums", "TestSub"]}}, "--expect", "pass")
        self.assertEqual(rc, 0, out)
        self.assertIn("redgreen: 3 of 3 pass as expected", out)
        self.assertIn("--- PASS: TestSub", (self.out / "root.out").read_text())

    def test_each_package_output_has_a_visible_name(self) -> None:
        self.assertEqual(redgreen.output_name("./llmprovider/providers/chatcompat/"),
                         "llmprovider_providers_chatcompat.out")
        self.assertEqual(redgreen.output_name("./wizard/"), "wizard.out")
        self.assertEqual(redgreen.output_name("."), "root.out")

    def test_red_at_the_first_commit_with_a_drop(self) -> None:
        rc, out = self.run_spec({"tests": {".": ["TestAddSums"]}, "files": ["fix_test.go"], "drop": ["TestSub"]},
                                "--expect", "fail", "--rev", self.first)
        self.assertEqual(rc, 0, out)
        self.assertIn("ok  fail TestAddSums", out)

    def test_green_expected_where_red_is_a_finding(self) -> None:
        rc, out = self.run_spec({"tests": {".": ["TestAddSums"]}, "files": ["fix_test.go"], "drop": ["TestSub"]},
                                "--expect", "pass", "--rev", self.first)
        self.assertEqual(rc, 1, out)
        self.assertIn("BAD fail TestAddSums", out)

    def test_red_expected_where_green_is_a_finding(self) -> None:
        rc, out = self.run_spec({"tests": {".": ["TestZero", "TestAddSums"]}}, "--expect", "fail")
        self.assertEqual(rc, 1, out)
        self.assertIn("BAD pass TestZero", out)

    def test_a_package_that_does_not_build_is_not_red(self) -> None:
        rc, out = self.run_spec({"tests": {".": ["TestAddSums"]}, "files": ["fix_test.go"]},
                                "--expect", "fail", "--rev", self.first)
        self.assertEqual(rc, 1, out)
        self.assertIn("did not build", out)
        self.assertIn("BAD missing (did not build) TestAddSums", out)

    def test_a_mistyped_name_is_missing(self) -> None:
        rc, out = self.run_spec({"tests": {".": ["TestZero", "TestAddSum"]}}, "--expect", "pass")
        self.assertEqual(rc, 1, out)
        self.assertIn("BAD missing TestAddSum", out)
        self.assertIn("ok  pass TestZero", out)

    def test_a_drop_that_matches_nothing_is_a_tool_error(self) -> None:
        rc, out = self.run_spec({"tests": {".": ["TestAddSums"]}, "files": ["fix_test.go"], "drop": ["TestNope"]},
                                "--expect", "fail", "--rev", self.first)
        self.assertEqual(rc, 2, out)
        self.assertIn("drop TestNope: 0 function(s) removed", out)

    def test_files_without_rev_is_a_tool_error(self) -> None:
        rc, out = self.run_spec({"tests": {".": ["TestZero"]}, "files": ["fix_test.go"]}, "--expect", "pass")
        self.assertEqual(rc, 2, out)
        self.assertIn('"files" and "drop" need --rev', out)

    def test_the_tree_is_not_written(self) -> None:
        before = (self.root / "fix_test.go").read_text()
        self.run_spec({"tests": {".": ["TestAddSums"]}, "files": ["fix_test.go"], "drop": ["TestSub"]},
                      "--expect", "fail", "--rev", self.first)
        self.assertEqual((self.root / "fix_test.go").read_text(), before)
        status = subprocess.run(["git", "status", "--porcelain"], cwd=self.root, capture_output=True, text=True,
                                check=True).stdout
        self.assertEqual(status, "")

    def test_a_drop_takes_the_comment_lines_above_it(self) -> None:
        text, n = redgreen.drop_func(RG_FIX_TEST, "TestSub")
        self.assertEqual(n, 1)
        self.assertNotIn("TestSub", text)
        self.assertNotIn("A second comment line", text)
        self.assertIn("// TestAddSums is red before the fix.", text)

    def test_a_subtest_line_does_not_count_for_its_parent(self) -> None:
        self.assertEqual(redgreen.results("    --- PASS: TestA/sub (0.00s)\n", ["TestA"]), {"TestA": "missing"})
        run = "=== RUN   TestA\n    --- PASS: TestA/sub (0.00s)\n--- FAIL: TestA (0.00s)\n"
        self.assertEqual(redgreen.results(run, ["TestA"]), {"TestA": "fail"})


LF_MOD = "module example.com/leak\n\ngo 1.21\n"
LF_GUARD = ("// Package localonly stands in for the real guard: Main prints its report when a test called Reach.\n"
            "package localonly\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"strings\"\n\t\"sync\"\n)\n\n"
            "var (\n\tmu    sync.Mutex\n\thosts []string\n)\n\n"
            "// Reach records a request to host.\nfunc Reach(host string) {\n\tmu.Lock()\n\thosts = append(hosts, host)\n"
            "\tmu.Unlock()\n}\n\n"
            "// Main runs m, and prints the real guard's report when a host was reached.\n"
            "func Main(m interface{ Run() int }) int {\n\tcode := m.Run()\n\tif len(hosts) > 0 {\n"
            "\t\tfmt.Fprintln(os.Stderr, \"localonly: the tests sent requests off the machine, to \"+"
            "strings.Join(hosts, \", \")+\"; a stand-in\")\n\t\treturn 1\n\t}\n\treturn code\n}\n")
LF_LEAKY = ("package leaky\n\nimport (\n\t\"os\"\n\t\"testing\"\n\n\t\"example.com/leak/internal/localonly\"\n)\n\n"
            "func TestMain(m *testing.M) { os.Exit(localonly.Main(m)) }\n\nvar a, b bool\n\n"
            "func TestClean1(t *testing.T) {}\n\n"
            "func TestAlone(t *testing.T) { localonly.Reach(\"alone.test:443 (1)\") }\n\n"
            "func TestClean2(t *testing.T) {}\n\n"
            "func TestPairA(t *testing.T) {\n\ta = true\n\tif b {\n\t\tlocalonly.Reach(\"pair.test:443 (1)\")\n\t}\n}\n\n"
            "func TestPairB(t *testing.T) {\n\tb = true\n\tif a {\n\t\tlocalonly.Reach(\"pair.test:443 (1)\")\n\t}\n}\n\n"
            "func TestClean3(t *testing.T) {}\n")
LF_CLEAN = ("package clean\n\nimport (\n\t\"os\"\n\t\"testing\"\n\n\t\"example.com/leak/internal/localonly\"\n)\n\n"
            "func TestMain(m *testing.M) { os.Exit(localonly.Main(m)) }\n\nfunc TestFine(t *testing.T) {}\n")
LF_NOGUARD = "package noguard\n\nimport \"testing\"\n\nfunc TestFine(t *testing.T) {}\n"
LF_SPLIT = ("package split\n\nimport (\n\t\"os\"\n\t\"testing\"\n\n\t\"example.com/leak/internal/localonly\"\n)\n\n"
            "func TestMain(m *testing.M) { os.Exit(localonly.Main(m)) }\n\nvar first bool\n\n"
            "func TestFirst(t *testing.T) { first = true }\n\n"
            "func TestSecond(t *testing.T) {\n\tif first {\n\t\tlocalonly.Reach(\"split.test:443 (1)\")\n\t}\n}\n")


@unittest.skipUnless(shutil.which("go"), "go is not on PATH")
class TestLeakFind(unittest.TestCase):
    """0067-MADR D2: a module with a stand-in guard, a test that leaks alone, and two that leak only together."""

    def setUp(self) -> None:
        self.root = temp_repo(self)
        files = {"go.mod": LF_MOD, "internal/localonly/localonly.go": LF_GUARD, "leaky/leaky_test.go": LF_LEAKY,
                 "clean/clean_test.go": LF_CLEAN, "noguard/noguard_test.go": LF_NOGUARD,
                 "split/split_test.go": LF_SPLIT}
        for rel, text in files.items():
            (self.root / rel).parent.mkdir(parents=True, exist_ok=True)
            (self.root / rel).write_text(text)
        git(self.root, "add", ".")
        git(self.root, "commit", "-q", "-m", "seed")

    def find(self, *argv: str) -> tuple[int, str]:
        return call(leakfind.main, "--root", str(self.root), "--runs", "1", *argv)

    def test_a_lone_leak_and_a_pair_are_named(self) -> None:
        rc, out = self.find("--chunk", "3", "./leaky/")
        self.assertEqual(rc, 1, out)
        self.assertIn("leakfind: 6 tests in ./leaky/", out)
        self.assertIn("LEAK TestAlone, alone: alone.test:443 (1)", out)
        self.assertIn("LEAK TestPairA, TestPairB, together: pair.test:443 (1)", out)
        self.assertNotIn("TestClean3,", out)
        self.assertIn("leakfind: 2 leak(s) found", out)

    def test_a_leak_across_chunks_is_not_clean(self) -> None:
        rc, out = self.find("--chunk", "1", "./split/")
        self.assertEqual(rc, 1, out)
        self.assertIn("whole package: leaks to split.test:443 (1)", out)
        self.assertIn("chunk 1 (1 tests): clean", out)
        self.assertIn("chunk 2 (1 tests): clean", out)
        self.assertIn("leaks whole but no chunk of 1 does; run with a larger --chunk", out)

    def test_a_package_with_no_leak_is_clean(self) -> None:
        rc, out = self.find("./clean/")
        self.assertEqual(rc, 0, out)
        self.assertIn("leakfind: no leak", out)

    def test_a_package_without_the_guard_is_a_tool_error(self) -> None:
        rc, out = self.find("./noguard/")
        self.assertEqual(rc, 2, out)
        self.assertIn("calls localonly.Main", out)

    def test_ssh_needs_dir_and_a_plain_alias(self) -> None:
        with mock.patch.object(leakfind, "run", side_effect=AssertionError("a refused --ssh ran a command")):
            for argv in (["--ssh", "devhost.test"], ["--ssh=-oProxyCommand=x", "--dir", "d"]):
                rc, out = self.find(*argv, "./clean/")
                self.assertEqual(rc, 2, out)
                self.assertIn("--ssh takes a host alias and needs --dir", out)

    def test_the_tree_is_not_written(self) -> None:
        self.find("--chunk", "3", "./leaky/")
        status = subprocess.run(["git", "status", "--porcelain"], cwd=self.root, capture_output=True, text=True,
                                check=True).stdout
        self.assertEqual(status, "")


class TestLeakFindParts(unittest.TestCase):
    def test_the_report_is_the_guards(self) -> None:
        source = (_common.ROOT / "internal/localonly/localonly.go").read_text(encoding="utf-8")
        self.assertIn('"' + leakfind.REPORT + '"', source)

    def test_the_remote_command_quotes_the_directory(self) -> None:
        cmd = leakfind.ssh_command("devhost.test", "work dir/sdk", ["go", "test", "-run", "^(TestA|TestB)$", "./x/"])
        self.assertEqual(cmd[:6], ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "devhost.test"])
        self.assertEqual(cmd[6], "cd 'work dir/sdk' && go test -run '^(TestA|TestB)$' ./x/")

    def test_only_test_names_leave_the_listing(self) -> None:
        listing = "TestA\nExampleB\nFuzzC\nBenchmarkD\nTestE; rm -rf x\nok  \texample.com/leak\t0.1s\n"
        self.assertEqual(leakfind.names_from_list(listing), ["TestA", "ExampleB", "FuzzC"])

    def test_a_runner_sees_every_set_through_the_pattern(self) -> None:
        seen: list[list[str]] = []

        def go(args: list[str]) -> str:
            seen.append(args)
            return "TestA\nTestB\n" if "-list" in args else ""
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            rc = leakfind.search(leakfind.Finder(go, "./x/", 1), 25)
        self.assertEqual(rc, 0, out.getvalue())
        self.assertIn("leakfind: no leak", out.getvalue())
        self.assertEqual(seen[1], ["go", "test", "-count=1", "-run", "^(TestA|TestB)$", "./x/"])


class TestApidiffArgs(unittest.TestCase):
    def test_the_pin_is_read_from_check_api(self) -> None:
        source = (_common.ROOT / "scripts" / "check_api.py").read_text()
        self.assertIn(f'APIDIFF = "{apidiff.pinned()}"', source)
        self.assertNotIn("golang.org/x/exp/cmd/apidiff@", (_common.ROOT / "tools" / "apidiff.py").read_text())

    def test_the_subtrees_are_read_from_check_api(self) -> None:
        """One run per subtree, never one for the module: the pattern
        `<module>/...` also matches the nested module authcontract
        (0074-PLAN Phase 4)."""
        self.assertEqual(apidiff.subtrees(), ["llmprovider", "wizard"])

    def test_merge_joins_the_reports_by_section(self) -> None:
        first = "Incompatible changes:\n- ./llmprovider.A: removed\nCompatible changes:\n- ./llmprovider.B: added\n"
        second = "Compatible changes:\n- ./wizard.C: added\n"
        third = "Incompatible changes:\n- ./wizard.D: removed\n"
        self.assertEqual(
            apidiff.merge([first, second, third]),
            "Incompatible changes:\n- ./llmprovider.A: removed\n- ./wizard.D: removed\n"
            "Compatible changes:\n- ./llmprovider.B: added\n- ./wizard.C: added\n")
        self.assertEqual(apidiff.merge(["", ""]), "")


LIVE_RUN = """=== RUN   TestLive_A
    live_a_test.go:10: answered
--- PASS: TestLive_A (0.10s)
=== RUN   TestLive_B
    live_b_test.go:20: set LLMPROVIDER_LIVE_X to run it
--- SKIP: TestLive_B (0.00s)
=== RUN   TestLive_C
=== RUN   TestLive_C/zen
    live_c_test.go:30: gateway transient: HTTP 402
=== RUN   TestLive_C/go
--- PASS: TestLive_C (0.50s)
    --- SKIP: TestLive_C/zen (0.10s)
    --- PASS: TestLive_C/go (0.40s)
ok  \texample.com/m\t1.0s
"""


class TestLiveSkips(unittest.TestCase):
    def classify(self, text: str) -> tuple[int, str]:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "run.txt"
            path.write_text(text)
            return call(liveskips.main, str(path))

    def test_counts_and_reasons(self) -> None:
        rc, out = self.classify(LIVE_RUN)
        self.assertEqual(rc, 0, out)
        self.assertIn("top level: 2 passed, 1 skipped, 0 failed", out)
        self.assertIn("subtests: 1 passed, 1 skipped, 0 failed", out)
        self.assertIn("SKIP TestLive_B: set LLMPROVIDER_LIVE_X to run it", out)
        self.assertIn("SKIP TestLive_C/zen: gateway transient: HTTP 402", out)

    def test_a_failure_exits_one(self) -> None:
        failed = LIVE_RUN.replace("--- PASS: TestLive_A", "--- FAIL: TestLive_A")
        rc, out = self.classify(failed)
        self.assertEqual(rc, 1, out)
        self.assertIn("top level: 1 passed, 1 skipped, 1 failed", out)
        self.assertIn("FAIL TestLive_A: answered", out)


class TestConsumerSource(unittest.TestCase):
    def test_imports_every_public_package(self) -> None:
        listed = _common.run(["go", "list", "./..."], check=True).stdout.split()
        public = [p for p in listed if "/internal" not in p]
        source = consumer.main_source(consumer.public_packages())
        for pkg in public:
            self.assertIn(f'_ "{pkg}"', source)
        self.assertNotIn("/internal", source)

    def test_builds_against_the_tree_with_no_network(self) -> None:
        with tempfile.TemporaryDirectory() as tmp, mock.patch.dict(os.environ, {"GOPROXY": "off", "GOFLAGS": "-mod=mod"}):
            rc, out = call(consumer.main, "--replace", "--platforms", "host", "--out", tmp)
        self.assertEqual(rc, 0, out)
        self.assertIn("go build", out)
        self.assertIn("consumer: every step passed", out)


class TestOffline(unittest.TestCase):
    """0035-MADR D5: a ModelInfo read with no request, and the plants that prove it."""

    NO_NETWORK = {"GOPROXY": "off", "GOFLAGS": "-mod=mod"}

    def run_offline(self, plants: dict[str, tuple[str, str]] | None = None) -> tuple[int, str]:
        with tempfile.TemporaryDirectory() as tmp, mock.patch.dict(os.environ, self.NO_NETWORK):
            out = io.StringIO()
            with contextlib.redirect_stdout(out), contextlib.redirect_stderr(out):
                rc = offline.main(["--replace", "--platforms", "host", "--out", tmp], plants=plants)
        return rc, out.getvalue()

    def test_each_plant_matches_the_program_once(self) -> None:
        for name, (old, _) in offline.PLANTS.items():
            self.assertEqual(offline.PROGRAM.count(old), 1, name)

    def test_passes_against_the_tree_with_no_network(self) -> None:
        rc, out = self.run_offline()
        self.assertEqual(rc, 0, out)
        self.assertIn("requests=0", out)
        for name in offline.PLANTS:
            self.assertIn(f"caught: {name}", out)
        self.assertIn("offline: every step passed", out)

    def test_a_harmless_plant_is_not_caught(self) -> None:
        rc, out = self.run_offline({"a comment": ("// Command offline", "// Command  offline")})
        self.assertEqual(rc, 1, out)
        self.assertIn("NOT CAUGHT: a comment", out)

    def test_an_unmatched_plant_is_a_tool_error(self) -> None:
        rc, out = self.run_offline({"absent": ("no such text", "")})
        self.assertEqual(rc, 2, out)
        self.assertIn('"absent"', out)
        self.assertNotIn("go get", out)
        self.assertNotIn("go mod", out)


class TestCacheCensus(unittest.TestCase):
    """0058-MADR D5: the tests that leave a loopback URL in the process's caches."""

    MODULE = "example.com/m"
    LOG = [
        "CENSUS metadata http://127.0.0.1:1/api.json",
        "=== RUN   TestA",
        "CENSUS listing http://127.0.0.1:2",
        "=== RUN   TestA/sub",
        "CENSUS probes http://127.0.0.1:2",
        "--- PASS: TestA (0.00s)",
        "=== RUN   TestB",
        "=== PAUSE TestB",
        "=== RUN   TestC",
        "=== CONT  TestB",
        "CENSUS metadata http://127.0.0.1:3/api.json",
        "ok  \texample.com/m/one\t0.1s",
        "CENSUS listed http://127.0.0.1:4",
        "=== RUN   TestD",
        "--- FAIL: TestD (0.00s)",
        "FAIL\texample.com/m/two\t0.1s",
    ]

    def test_each_anchor_matches_the_tree_once(self) -> None:
        self.assertEqual(cachecensus.check_anchors(_common.ROOT, cachecensus.INSTRUMENTS), [])

    def test_lines_go_to_their_package_and_top_level_test(self) -> None:
        self.assertEqual(cachecensus.attribute(self.LOG, self.MODULE), {
            "one": {"?": ["metadata"], "TestA": ["listing", "probes"], "TestB": ["metadata"]},
            "two": {"?": ["listed"]},
        })

    def test_an_anchor_found_twice_is_named(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            (Path(tmp) / "a.go").write_text("x()\nx()\ny()\n", encoding="utf-8")
            bad = cachecensus.check_anchors(Path(tmp), [("a.go", "x()\n", ""), ("a.go", "y()\n", "")])
        self.assertEqual(bad, ["a.go: anchor found 2 times: 'x()'"])

    def test_a_missing_anchor_is_a_tool_error_before_any_clone(self) -> None:
        missing = [("llmprovider/catalog/cache.go", "no such anchor\n", "")]
        with tempfile.TemporaryDirectory() as tmp:
            out = io.StringIO()
            with contextlib.redirect_stdout(out), contextlib.redirect_stderr(out):
                rc = cachecensus.main(["--out", tmp], instruments=missing)
            self.assertEqual(rc, 2, out.getvalue())
            self.assertIn("llmprovider/catalog/cache.go: anchor found 0 times", out.getvalue())
            self.assertFalse((Path(tmp) / "clone").exists())

    # 0058-MADR's amendment of 2026-10-24, D7: --any-url.
    SHARED_LOG = [
        "=== RUN   TestA",
        "CENSUS listing claude|https://api.anthropic.com||fp1",
        "CENSUS listed claude|https://api.anthropic.com||fp1",
        "=== RUN   TestB",
        "CENSUS listing claude|https://api.anthropic.com||fp1",
        "CENSUS listing claude|https://api.anthropic.com||fp2",
        "ok  \texample.com/m/one\t0.1s",
        "=== RUN   TestC",
        "CENSUS listing claude|https://api.anthropic.com||fp1",
        "ok  \texample.com/m/two\t0.1s",
    ]

    def test_any_url_uses_the_same_anchors(self) -> None:
        self.assertEqual([(r, o) for r, o, _ in cachecensus.ANY_URL_INSTRUMENTS],
                         [(r, o) for r, o, _ in cachecensus.INSTRUMENTS])
        self.assertEqual(cachecensus.check_anchors(_common.ROOT, cachecensus.ANY_URL_INSTRUMENTS), [])

    def test_any_url_reports_each_listing_by_its_whole_key(self) -> None:
        whole = [n for _, _, n in cachecensus.ANY_URL_INSTRUMENTS if "censusKey(key)" in n]
        self.assertEqual(len(whole), 3, "the listing, the listed set and the show")
        self.assertFalse(any("key.baseURL" in n for _, _, n in cachecensus.ANY_URL_INSTRUMENTS))
        self.assertIn("k.credential", cachecensus.ANY_URL_HELPERS["llmprovider/catalog/zz_cachecensus.go"])

    def test_any_url_has_no_loopback_filter(self) -> None:
        for rel, text in cachecensus.ANY_URL_HELPERS.items():
            self.assertNotIn("127.0.0.1", text, rel)
        for rel, text in cachecensus.HELPERS.items():
            self.assertIn("127.0.0.1", text, rel)

    def test_a_key_two_tests_wrote_is_reported(self) -> None:
        self.assertEqual(cachecensus.shared_keys(self.SHARED_LOG, self.MODULE), {
            "one": {"listing claude|https://api.anthropic.com||fp1": ["TestA", "TestB"]},
        })


class TestClaims(unittest.TestCase):
    """0040-PLAN Phase 0: each evidence row holds its text, on its line, at its revision."""

    SOURCE = "package demo\n\nfunc Alpha() {}\n\nfunc Beta() string { return \"a|b\" }\n"

    def setUp(self) -> None:
        self.repo = temp_repo(self)
        (self.repo / "demo.go").write_text(self.SOURCE, encoding="utf-8")
        git(self.repo, "add", "demo.go")
        git(self.repo, "commit", "-q", "-m", "one")
        self.rev = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=self.repo, check=True,
                                  capture_output=True, text=True).stdout.strip()
        # A later commit moves Alpha; the claims stay pinned to the first.
        (self.repo / "demo.go").write_text("package demo\n\n\n\n" + self.SOURCE[len("package demo\n\n"):], encoding="utf-8")
        git(self.repo, "commit", "-q", "-am", "two")

    def record(self, rows: list[str], marker: str | None = None) -> Path:
        marker = f"<!-- claims: self={self.rev} -->" if marker is None else marker
        body = "# R\n\n" + marker + "\n\n| Id | Claim | Where | Matched |\n| :--- | :--- | :--- | :--- |\n"
        path = self.repo / "record.md"
        path.write_text(body + "".join(r + "\n" for r in rows), encoding="utf-8")
        return path

    def claims(self, *argv: str) -> tuple[int, str]:
        return call(claims.main, *argv, "--repo", f"self={self.repo}")

    def test_passing_rows(self) -> None:
        rec = self.record(["| E1 | alpha | `demo.go:3` | `func Alpha() {}` |",
                           "| P2 | an escaped pipe | `demo.go:5` | `return \"a\\|b\"` |"])
        rc, out = self.claims(str(rec))
        self.assertEqual(rc, 0, out)
        self.assertIn("2 of 2 pass", out)

    def test_pinned_revision_not_head(self) -> None:
        # At HEAD Alpha is on line 5; the record's revision has it on line 3.
        rc, out = self.claims(str(self.record(["| E1 | alpha | `demo.go:3` | `func Alpha() {}` |"])))
        self.assertEqual(rc, 0, out)

    def test_wrong_line_reports_where_found_and_writes_a_fix(self) -> None:
        rec = self.record(["| E1 | alpha | `demo.go:4` | `func Alpha() {}` |"])
        spec = self.repo / "fix.json"
        rc, out = self.claims(str(rec), "--fixes", str(spec))
        self.assertEqual(rc, 1, out)
        self.assertIn("FAIL E1 demo.go:4", out)
        self.assertIn("text found at [3]", out)
        fixes = json.loads(spec.read_text(encoding="utf-8"))
        self.assertEqual(len(fixes), 1)
        self.assertIn("`demo.go:3`", fixes[0]["new"])
        self.assertEqual(rec.read_text(encoding="utf-8").count(fixes[0]["old"]), 1)

    def test_missing_file(self) -> None:
        rc, out = self.claims(str(self.record(["| E1 | gone | `nope.go:1` | `x` |"])))
        self.assertEqual(rc, 1, out)
        self.assertIn("FAIL E1 nope.go:1", out)

    def test_revision_that_lacks_the_file(self) -> None:
        rec = self.record(["| E1 | alpha | `demo.go:3` | `func Alpha() {}` |"], marker="<!-- claims: self=0000000 -->")
        rc, out = self.claims(str(rec))
        self.assertEqual(rc, 1, out)
        self.assertIn("FAIL E1", out)

    def test_no_rows_is_a_finding(self) -> None:
        rc, out = self.claims(str(self.record([])))
        self.assertEqual(rc, 1, out)
        self.assertIn("no evidence rows", out)

    def test_no_revision_names_the_fix(self) -> None:
        rc, out = self.claims(str(self.record(["| E1 | alpha | `demo.go:3` | `func Alpha() {}` |"], marker="")))
        self.assertEqual(rc, 1, out)
        self.assertIn("--rev self=REV", out)

    def plants(self, plants: list[dict]) -> tuple[int, str]:
        self.record(["| E1 | alpha | `demo.go:3` | `func Alpha() {}` |",
                     "| E2 | beta | `demo.go:5` | `func Beta() string` |"])
        spec = self.repo / "plants.json"
        spec.write_text(json.dumps({"record": "record.md", "plants": plants}), encoding="utf-8")
        return self.claims("--plants", str(spec))

    def test_plants_a_moved_line_is_caught(self) -> None:
        rc, out = self.plants([{"name": "E1 off by one", "old": "`demo.go:3`", "new": "`demo.go:4`"},
                               {"name": "the marker's revision", "old": f"self={self.rev}", "new": "self=0000000"}])
        self.assertEqual(rc, 0, out)
        self.assertIn("== E1 off by one: caught (FAIL E1 demo.go:4", out)
        self.assertIn("all 2 plant(s) caught", out)
        self.assertIn("`demo.go:3`", (self.repo / "record.md").read_text(encoding="utf-8"),
                      "the record itself was changed")

    def test_plants_a_harmless_plant_is_a_finding(self) -> None:
        rc, out = self.plants([{"name": "the claim's words", "old": "| alpha |", "new": "| ALPHA |"}])
        self.assertEqual(rc, 1, out)
        self.assertIn("== the claim's words: NOT CAUGHT", out)

    def test_plants_an_old_matching_twice_is_a_tool_error(self) -> None:
        rc, out = self.plants([{"name": "ambiguous", "old": "demo.go", "new": "x.go"}])
        self.assertEqual(rc, 2, out)
        self.assertIn("ambiguous", out)

    def test_plants_from_files(self) -> None:
        (self.repo / "old.txt").write_text("`demo.go:5`", encoding="utf-8")
        (self.repo / "new.txt").write_text("`demo.go:6`", encoding="utf-8")
        rc, out = self.plants([{"name": "E2 moved", "old_file": "old.txt", "new_file": "new.txt"}])
        self.assertEqual(rc, 0, out)
        self.assertIn("== E2 moved: caught", out)

    def test_plants_on_a_failing_record_are_refused(self) -> None:
        self.record(["| E1 | alpha | `demo.go:4` | `func Alpha() {}` |"])
        spec = self.repo / "plants.json"
        spec.write_text(json.dumps({"record": "record.md", "plants": [
            {"name": "would look caught", "old": "| alpha |", "new": "| ALPHA |"}]}), encoding="utf-8")
        rc, out = self.claims("--plants", str(spec))
        self.assertEqual(rc, 1, out)
        self.assertIn("does not pass unplanted", out)
        self.assertNotIn("would look caught", out)

    def test_other_repository_by_search_and_missing(self) -> None:
        base = self.repo.parent / f"claims-search-{self.rev}"
        base.mkdir()
        self.addCleanup(shutil.rmtree, base, True)
        other = base / "other-repo"
        other.mkdir()
        git(other, "init", "-q")
        git(other, "config", "user.email", "tools-test@invalid")
        git(other, "config", "user.name", "tools-test")
        (other / "x.txt").write_text("first\nsecond\n", encoding="utf-8")
        git(other, "add", "x.txt")
        git(other, "commit", "-q", "-m", "x")
        orev = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=other, check=True,
                              capture_output=True, text=True).stdout.strip()
        rec = self.record(["| E1 | other | `other-repo:x.txt:2` | `second` |"],
                          marker=f"<!-- claims: self={self.rev} other-repo={orev} -->")
        rc, out = self.claims(str(rec), "--search", str(base))
        self.assertEqual(rc, 0, out)
        rc, out = self.claims(str(rec), "--search", str(self.repo))
        self.assertEqual(rc, 1, out)
        self.assertIn("--repo other-repo=DIR", out)

    def test_json_output(self) -> None:
        rec = self.record(["| E1 | alpha | `demo.go:3` | `func Alpha() {}` |",
                           "| E2 | wrong | `demo.go:1` | `func Alpha() {}` |"])
        rc, out = self.claims(str(rec), "--json")
        self.assertEqual(rc, 1, out)
        data = json.loads(out)
        self.assertFalse(data["ok"])
        rows = {r["id"]: r for r in data["records"][0]["rows"]}
        self.assertEqual(rows["E1"]["status"], "pass")
        self.assertEqual(rows["E2"]["found"], [3])

    def test_double_backtick_cell(self) -> None:
        (self.repo / "tick.go").write_text("x := `raw`\n", encoding="utf-8")
        git(self.repo, "add", "tick.go")
        git(self.repo, "commit", "-q", "-m", "tick")
        rev = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=self.repo, check=True,
                             capture_output=True, text=True).stdout.strip()
        rec = self.record(["| E1 | tick | `tick.go:1` | `` x := `raw` `` |"], marker=f"<!-- claims: self={rev} -->")
        rc, out = self.claims(str(rec))
        self.assertEqual(rc, 0, out)

    def test_marker_quoted_in_prose_is_not_the_marker(self) -> None:
        marker = f"Write `<!-- claims: NAME=REV ... -->` to name them.\n\n<!-- claims: self={self.rev} -->"
        rc, out = self.claims(str(self.record(["| E1 | alpha | `demo.go:3` | `func Alpha() {}` |"], marker=marker)))
        self.assertEqual(rc, 0, out)

    def test_bad_arguments_are_a_tool_error(self) -> None:
        rc, _ = call(claims.main)
        self.assertEqual(rc, 2)
        rc, _ = call(claims.main, "x.md", "--repo", "noequals")
        self.assertEqual(rc, 2)


class TestCatalogCheck(unittest.TestCase):
    """0041-PLAN Phase 7: a changed six is reported; a clean snapshot is not; no rule is (0062-MADR D8)."""

    @classmethod
    def setUpClass(cls) -> None:
        cls.tmp = Path(tempfile.mkdtemp())
        base = catalogcheck.latest_snapshot()
        assert base is not None
        cls.clean = cls.tmp / "clean"
        shutil.copytree(base, cls.clean)
        cls.planted = cls.tmp / "planted"
        shutil.copytree(base, cls.planted)
        # A changed six: a model of Kilo's utility six leaves the listing.
        kilo = cls.planted / "kilo.json"
        listing = json.loads(kilo.read_text(encoding="utf-8"))
        listing["data"] = [m for m in listing["data"] if m["id"] != "deepseek/deepseek-v4.1-flash"]
        kilo.write_text(json.dumps(listing), encoding="utf-8")

    @classmethod
    def tearDownClass(cls) -> None:
        shutil.rmtree(cls.tmp, ignore_errors=True)

    def test_a_clean_snapshot_reports_nothing(self) -> None:
        rc, out = call(catalogcheck.main, "--dir", str(self.clean))
        self.assertEqual(rc, 0, out)
        self.assertIn(": 0 changed", out)
        self.assertNotIn("rules:", out)

    def test_a_planted_snapshot_reports_both(self) -> None:
        rc, out = call(catalogcheck.main, "--dir", str(self.planted))
        self.assertEqual(rc, 1, out)
        self.assertNotIn("rules:", out)
        self.assertRegex(out, r"kilo utility: \d of 6 changed: .*-deepseek/deepseek-v4\.1-flash")

    def test_a_dir_without_a_snapshot_is_a_tool_error(self) -> None:
        rc, out = call(catalogcheck.main, "--dir", str(self.tmp))
        self.assertEqual(rc, 2, out)

    def test_an_existing_out_is_a_tool_error(self) -> None:
        rc, out = call(catalogcheck.main, "--out", str(self.clean))
        self.assertEqual(rc, 2, out)
        self.assertIn("the catalogs command failed", out)


if __name__ == "__main__":
    unittest.main()
