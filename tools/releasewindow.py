#!/usr/bin/env python3
"""releasewindow: the changelog self-tests hold in each release state, not only in the tree's.

Defined by docs/decisions/0094-MADR-refuse-an-edit-to-a-released-changelog-section.md
RS17, RS33, RS34 and RS35 (0094-PLAN Phase 2 step 1, deviation D-5). The
changelog cases of scripts/test_gates.py must not depend on which tag is
newest. This tool makes three scratch clones of the working tree, each in one
release state, first tagging any section the tree has dated but not tagged:

  window  the next release's date step committed and untagged, and the nested
          module's date step too (RS33, RS34);
  tagged  the date step committed at its date and tagged;
  early   the next tag made before its date step, at a commit dated by it,
          then the date step committed (RS35; v1.24.0 was released so).

The next version is the newest release's minor version plus one, dated
today (UTC). A newest tag ahead of the first dated section (a tree in the
early state) has its section date-stepped at its tag's date first. In each
clone it runs the changelog cases (`-k changelog -k ChangelogReleased`) and
`make changelog-check`, then each plant of PLANTS that the state is for,
through tools/plant.py --root CLONE: a plant a state must catch fails it when
not caught, or when its anchor does not match (plant.py's exit 2, labelled
apart). Each failing check prints a FAIL line naming its state and check, and
a tool error prints one too. Nothing in the tree, or in --src, changes; each
clone is removed when its state finishes, unless --keep.

Usage:
  tools/releasewindow.py [--src DIR] [--out DIR] [--states window,tagged,early] [--no-plants] [--keep]

Each check's and each plant's output is in --out. Exit codes: 0 every check
passed, 1 a check failed, 2 tool error.
"""
from __future__ import annotations

import argparse
import datetime
import json
import re
import shutil
import subprocess
import sys
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from pathlib import Path

from _common import ROOT, out_dir, run, scratch_clone

STATES = ("window", "tagged", "early")
SUBSET = ("python3", "-B", "-m", "unittest", "scripts/test_gates.py", "-k", "changelog", "-k", "ChangelogReleased")
NESTED = "authcontract/CHANGELOG.md"
DATED = re.compile(r"^## \[(v(\d+)\.(\d+)\.(\d+))\] - (\d{4}-\d{2}-\d{2})", re.M)
VERSION = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")
UNRELEASED = "## [Unreleased]\n"
COMPARE = re.compile(r"^\[Unreleased\]: (\S+/compare/)(v\d+\.\d+\.\d+)\.\.\.HEAD$", re.M)
IDENT = {"GIT_AUTHOR_NAME": "tools", "GIT_AUTHOR_EMAIL": "tools@invalid",
         "GIT_COMMITTER_NAME": "tools", "GIT_COMMITTER_EMAIL": "tools@invalid"}
RS3_PICK = ("                       if tag_exists(self.tree, m.group(1)) and carried_at(self.tree, m.group(1))), "
            "None)\n")


@dataclass(frozen=True)
class Plant:
    """One change to scripts/test_gates.py, and the states that must catch it."""
    name: str
    old: str
    new: str
    must: tuple[str, ...]
    on: tuple[str, ...]


PLANTS = (
    Plant("RS3's case takes the first release section, tagged or not", RS3_PICK,
          "                       if True), None)\n", ("window",), ("window", "tagged")),
    Plant("close_window made a no-op",
          "    if first is not None and not tag_exists(tree, first.group(1)):\n"
          "        tag_dated(tree, first.group(1), first.group(2))\n", "    pass\n", ("window",), ("window", "tagged")),
    Plant("the copy's authcontract tags not kept (RS34)",
          "        if not tag_exists(tree, name):\n            subprocess.run([\"git\", \"tag\", name, obj]",
          "        if False:\n            subprocess.run([\"git\", \"tag\", name, obj]", ("window",), ("window",)),
    Plant("RS3's case without its carry check (RS35)", RS3_PICK,
          "                       if tag_exists(self.tree, m.group(1))), None)\n", ("early",), ("early",)),
)


class ToolError(Exception):
    """A clone this tool cannot put in its state: exit 2."""


def git(clone: Path, *args: str, date: str | None = None) -> str:
    env = dict(IDENT)
    if date is not None:
        env["GIT_AUTHOR_DATE"] = env["GIT_COMMITTER_DATE"] = f"{date}T12:00:00+0000"
    proc = run(["git", *args], cwd=clone, env=env)
    if proc.returncode != 0:
        raise ToolError(f"git {' '.join(args)}: {proc.stderr.strip()}")
    return proc.stdout.strip()


def tagged(clone: Path, tag: str) -> bool:
    return run(["git", "rev-parse", "-q", "--verify", f"refs/tags/{tag}"], cwd=clone).returncode == 0


def commit(clone: Path, message: str, date: str | None = None) -> str:
    """The working tree committed with `git commit-tree`, so no hook runs; HEAD moves to it."""
    git(clone, "add", "-A")
    sha = git(clone, "commit-tree", git(clone, "write-tree"), "-p", "HEAD", "-m", message, date=date)
    git(clone, "update-ref", "HEAD", sha)
    return sha


def newest_tag(clone: Path, prefix: str) -> tuple[str, tuple[int, int, int]] | None:
    """The newest prefix+vX.Y.Z tag's version name and number, or None."""
    found = [(name, (int(m.group(1)), int(m.group(2)), int(m.group(3))))
             for tag in git(clone, "tag", "--list", f"{prefix}v*").split()
             if (m := VERSION.match(name := tag.removeprefix(prefix)))]
    return max(found, key=lambda item: item[1]) if found else None


def close(clone: Path, path: str, prefix: str) -> tuple[int, int, int]:
    """Bring the file's newest release to a tagged section, and return its version.

    A newest tag ahead of the first dated section is a release whose tag was
    made before its date step (the early state): its section is date-stepped
    at its tag's commit date. Otherwise the first section, when it has no tag,
    is tagged at a commit dated by its heading."""
    first = DATED.search((clone / path).read_text(encoding="utf-8"))
    if first is None:
        raise ToolError(f"{path} has no dated release section")
    at = (int(first.group(2)), int(first.group(3)), int(first.group(4)))
    newest = newest_tag(clone, prefix)
    if newest is not None and newest[1] > at:
        name, number = newest
        date_step(clone, path, name, git(clone, "log", "-1", "--format=%cs", prefix + name))
        commit(clone, f"date step for {prefix}{name}")
        return number
    if not tagged(clone, prefix + first.group(1)):
        git(clone, "tag", prefix + first.group(1), commit(clone, f"tag {prefix}{first.group(1)}", first.group(5)))
    return at


def date_step(clone: Path, path: str, version: str, date: str) -> None:
    """[Unreleased]'s entries become the section of version; the compare links follow, where the file has them."""
    log = clone / path
    text = log.read_text(encoding="utf-8")
    if text.count(UNRELEASED) == 1:
        text = text.replace(UNRELEASED, f"{UNRELEASED}\n## [{version}] - {date}\n", 1)
    else:
        first = DATED.search(text)
        text = text[:first.start()] + f"## [{version}] - {date}\n\n" + text[first.start():]
    link = COMPARE.search(text)
    if link is not None:
        base, prev = link.group(1), link.group(2)
        text = text.replace(link.group(0), f"[Unreleased]: {base}{version}...HEAD\n[{version}]: {base}{prev}...{version}",
                            1)
    log.write_text(text, encoding="utf-8")


def make_state(clone: Path, state: str, today: str) -> str:
    """Put clone, a scratch clone of the tree, in state; return the next version."""
    major, minor, _ = close(clone, "CHANGELOG.md", "")
    nmajor, nminor, _ = close(clone, NESTED, "authcontract/")
    nxt, nested_next = f"v{major}.{minor + 1}.0", f"v{nmajor}.{nminor + 1}.0"
    if state == "window":
        date_step(clone, "CHANGELOG.md", nxt, today)
        date_step(clone, NESTED, nested_next, today)
        commit(clone, "date step")
    elif state == "tagged":
        date_step(clone, "CHANGELOG.md", nxt, today)
        git(clone, "tag", nxt, commit(clone, "date step", today))
    elif state == "early":
        git(clone, "tag", nxt, commit(clone, "tagged before its date step", today))
        date_step(clone, "CHANGELOG.md", nxt, today)
        commit(clone, "date step")
    else:
        raise ToolError(f"no state {state!r}")
    return nxt


def check(name: str, args: Sequence[str], cwd: Path, out: Path, failures: list[str], detail: str = "") -> bool:
    proc = run(args, cwd=cwd)
    path = out / (re.sub(r"[^A-Za-z0-9]+", "-", name).strip("-") + ".out")
    path.write_text(proc.stdout + proc.stderr, encoding="utf-8")
    if proc.returncode == 0:
        print(f"ok   {name}")
        return True
    line = f"FAIL {name}: exit {proc.returncode}{detail}; output in {path}"
    print(line)
    failures.append(line)
    return False


# plant.py's exit codes (tools/plant.py): 0 caught, 1 not caught, 2 an anchor
# that did not match once, or its tool error.
PLANT_EXIT = {0: "caught", 1: "not caught", 2: "its anchor did not match, or plant.py's tool error"}


def plant_check(state: str, n: int, p: Plant, clone: Path, out: Path, failures: list[str]) -> None:
    """Run one plant through tools/plant.py on clone, keep its output, and label its exit."""
    spec = out / f"{state}-plant-{n}.json"
    spec.write_text(json.dumps({"command": list(SUBSET), "plants": [
        {"name": p.name, "file": "scripts/test_gates.py", "old": p.old, "new": p.new}]}), encoding="utf-8")
    proc = run(("python3", "-B", str(ROOT / "tools/plant.py"), "--root", str(clone), str(spec)), cwd=ROOT)
    path = out / f"{state}-plant-{n}.out"
    path.write_text(proc.stdout + proc.stderr, encoding="utf-8")
    label = PLANT_EXIT.get(proc.returncode, "plant.py's tool error")
    name = f"{state} plant: {p.name}"
    if state not in p.must:
        print(f"note {name}: exit {proc.returncode} ({label}; not required); output in {path}")
    elif proc.returncode == 0:
        print(f"ok   {name}")
    else:
        line = f"FAIL {name}: exit {proc.returncode} ({label}); output in {path}"
        print(line)
        failures.append(line)


def run_state(src: Path, out: Path, state: str, plants: bool, today: str, keep: bool = False) -> list[str]:
    """One state's clone and its checks; the FAIL lines. The clone is removed
    when the state finishes, a ToolError included, unless keep."""
    clone = out / state
    if clone.exists():
        shutil.rmtree(clone)
    try:
        try:
            scratch_clone(src, clone)
        except subprocess.CalledProcessError as err:
            raise ToolError(f"{state}: cannot make the scratch clone of {src}: "
                            f"{(err.stderr or '').strip() or err}") from None
        nxt = make_state(clone, state, today)
        print(f"== {state}: next release {nxt} dated {today}")
        failures: list[str] = []
        check(f"{state} changelog cases", SUBSET, clone, out, failures)
        check(f"{state} make changelog-check", ("make", "changelog-check"), clone, out, failures)
        if plants:
            for n, p in enumerate(PLANTS):
                if state in p.on:
                    plant_check(state, n, p, clone, out, failures)
        return failures
    finally:
        if not keep:
            shutil.rmtree(clone, ignore_errors=True)


def main(argv: list[str] | None = None, *, today: Callable[[], str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--src", type=Path, default=ROOT, help="the tree to clone (default: this one)")
    parser.add_argument("--out", type=Path, default=None, help="where the clones and outputs go")
    parser.add_argument("--states", default=",".join(STATES), help="a comma-separated subset of " + ", ".join(STATES))
    parser.add_argument("--no-plants", action="store_true", help="run the cases and the gate only")
    parser.add_argument("--keep", action="store_true", help="keep each state's clone in --out, to debug it")
    try:
        args = parser.parse_args(argv)
    except SystemExit as exit_:
        return 0 if exit_.code == 0 else 2
    states = [s for s in args.states.split(",") if s]
    if not states or any(s not in STATES for s in states):
        print(f"releasewindow: --states takes {', '.join(STATES)}", file=sys.stderr)
        return 2
    out = args.out or out_dir("releasewindow")
    out.mkdir(parents=True, exist_ok=True)
    day = (today or (lambda: datetime.datetime.now(datetime.UTC).date().isoformat()))()
    failures: list[str] = []
    try:
        for state in states:
            failures += run_state(args.src.resolve(), out, state, not args.no_plants, day, args.keep)
    except (ToolError, OSError) as err:
        print(f"FAIL releasewindow: {err}")
        return 2
    print(f"releasewindow: {len(states)} state(s), {len(failures)} failure(s); output in {out}")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
