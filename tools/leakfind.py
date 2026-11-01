#!/usr/bin/env python3
"""leakfind: the tests behind a localonly leak, by chunks, one by one, and leaving out.

Defined by docs/decisions/0067-MADR-keep-four-more-scratch-helpers-as-tools.md
D2. internal/localonly's guard fails a package's run when its tests send a
request off the machine, and names each host, never the test (0065-MADR).
This tool names the tests:

1. the package's tests, by `go test -list`, run whole; a package that does
   not leak whole is clean;
2. run in chunks of --chunk; in each chunk that leaks, each test alone, and
   a test that leaks alone is reported;
3. when a chunk leaks and none of its tests leaks alone, each test is left
   out in turn; a test whose absence stops the leak is one the leak needs,
   and those tests are run together to confirm it.

A run leaks when its output holds the guard's report. Each set is run up to
--runs times and leaks if any run does, since a leak that depends on timing
does not show in every run; so "stops the leak" means in every run.

It runs on a scratch clone of the working tree; or, with --ssh ALIAS --dir
DIR, in the checkout DIR on the host ALIAS, for a leak that shows on one
platform only (0067-MADR Q2). It never copies to the host, and DIR must hold
the same package. The package must call localonly.Main in a _test.go file in
the tree.

Usage:
  tools/leakfind.py [--root DIR] [--chunk N] [--runs N] [--ssh ALIAS --dir DIR] PACKAGE

Exit codes: 0 no leak, 1 a leak found, 2 tool error.
"""
from __future__ import annotations

import argparse
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
from collections.abc import Callable
from pathlib import Path

from _common import ROOT, run, scratch_clone

# The text before the hosts in localonly.Report (internal/localonly/localonly.go);
# a test asserts it is still there.
REPORT = "localonly: the tests sent requests off the machine, to "
LEAK = re.compile(re.escape(REPORT) + r"([^;\n]+)")
GUARD = re.compile(r"localonly\.Main\(")
NAME = re.compile(r"^(Test|Example|Fuzz)\w+$")
PACKAGE = re.compile(r"^\.(/[\w.-]+)*/?$")
ALIAS = re.compile(r"^\w[\w.-]*$")

Runner = Callable[[list[str]], str]


def has_guard(root: Path, package: str) -> bool:
    """Whether a _test.go file in package's directory calls localonly.Main."""
    return any(GUARD.search(p.read_text(encoding="utf-8")) for p in (root / package).glob("*_test.go"))


def names_from_list(output: str) -> list[str]:
    """The test names `go test -list` printed; any other line is dropped, so no other text enters a pattern."""
    return [line.strip() for line in output.splitlines() if NAME.match(line.strip())]


def ssh_command(alias: str, directory: str, args: list[str]) -> list[str]:
    """The ssh command that runs args in directory on alias."""
    remote = "cd " + shlex.quote(directory) + " && " + " ".join(shlex.quote(a) for a in args)
    return ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", alias, remote]


def local_runner(clone: Path) -> Runner:
    def go(args: list[str]) -> str:
        p = run(args, cwd=clone)
        return p.stdout + p.stderr
    return go


def ssh_runner(alias: str, directory: str) -> Runner:
    def go(args: list[str]) -> str:
        p = run(ssh_command(alias, directory, args))
        return p.stdout + p.stderr
    return go


class Finder:
    """The search, over a runner, for one package."""

    def __init__(self, go: Runner, package: str, runs: int) -> None:
        self.go, self.package, self.runs = go, package, runs

    def leaks(self, names: list[str]) -> str:
        """The hosts the first leaking run of names reached, or "" if none of --runs runs leaked."""
        for _ in range(self.runs):
            out = self.go(["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", self.package])
            if m := LEAK.search(out):
                return m.group(1).strip()
        return ""

    def needed(self, chunk: list[str]) -> list[str]:
        """The tests of chunk whose absence stops its leak."""
        return [name for name in chunk if not self.leaks([n for n in chunk if n != name])]


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository (default: this one)")
    parser.add_argument("--chunk", type=int, default=25, help="tests per chunk (default: 25)")
    parser.add_argument("--runs", type=int, default=2, help="runs of each set before it is called clean (default: 2)")
    parser.add_argument("--ssh", default=None, help="run on this ssh host alias, in --dir")
    parser.add_argument("--dir", default=None, help="the checkout on the --ssh host")
    parser.add_argument("package", help="the package, as ./path/")
    args = parser.parse_args(argv)
    root = args.root.resolve()
    if not PACKAGE.match(args.package):
        print(f"leakfind: {args.package!r} is not a package path like ./wizard/", file=sys.stderr)
        return 2
    if (args.ssh is None) != (args.dir is None) or (args.ssh is not None and not ALIAS.match(args.ssh)):
        print("leakfind: --ssh takes a host alias and needs --dir", file=sys.stderr)
        return 2
    if args.chunk < 1 or args.runs < 1:
        print("leakfind: --chunk and --runs must be at least 1", file=sys.stderr)
        return 2
    if not (root / args.package).is_dir() or not has_guard(root, args.package):
        print(f"leakfind: no _test.go in {args.package} calls localonly.Main, so nothing reports a leak",
              file=sys.stderr)
        return 2

    work = Path(tempfile.mkdtemp(prefix="leakfind-"))
    try:
        if args.ssh is None:
            clone = work / "clone"
            try:
                scratch_clone(root, clone)
            except (OSError, subprocess.CalledProcessError) as err:
                print(f"leakfind: cannot make the scratch clone: {err}", file=sys.stderr)
                return 2
            go = local_runner(clone)
        else:
            go = ssh_runner(args.ssh, args.dir)
        return search(Finder(go, args.package, args.runs), args.chunk)
    except OSError as err:
        print(f"leakfind: cannot run: {err}", file=sys.stderr)
        return 2
    finally:
        shutil.rmtree(work, ignore_errors=True)


def search(f: Finder, size: int) -> int:
    listing = f.go(["go", "test", "-list", ".", f.package])
    tests = names_from_list(listing)
    if not tests:
        print(f"leakfind: go test -list found no tests in {f.package}:", file=sys.stderr)
        print(listing.strip()[-600:], file=sys.stderr)
        return 2
    print(f"leakfind: {len(tests)} tests in {f.package}", flush=True)
    whole = f.leaks(tests)
    if not whole:
        print("leakfind: no leak", flush=True)
        return 0
    print(f"whole package: leaks to {whole}", flush=True)
    found = 0
    for i in range(0, len(tests), size):
        chunk = tests[i:i + size]
        hosts = f.leaks(chunk)
        print(f"chunk {i // size + 1} ({len(chunk)} tests): {'leaks to ' + hosts if hosts else 'clean'}", flush=True)
        if not hosts:
            continue
        alone = [(name, h) for name in chunk if (h := f.leaks([name]))]
        for name, h in alone:
            print(f"  LEAK {name}, alone: {h}", flush=True)
        found += len(alone)
        if alone:
            continue
        needed = f.needed(chunk)
        together = f.leaks(needed) if needed else ""
        if together:
            print(f"  LEAK {', '.join(needed)}, together: {together}", flush=True)
        else:
            print(f"  LEAK the chunk as a whole; no one test's absence stops it: {', '.join(chunk)}", flush=True)
        found += 1
    if not found:
        print(f"leakfind: the package leaks whole but no chunk of {size} does; run with a larger --chunk", flush=True)
        return 1
    print(f"leakfind: {found} leak(s) found", flush=True)
    return 1


if __name__ == "__main__":
    sys.exit(main())
