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
import subprocess
import tempfile
import unittest
from pathlib import Path

import _common
import denyscan
import edit
import linkcheck


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


if __name__ == "__main__":
    unittest.main()
