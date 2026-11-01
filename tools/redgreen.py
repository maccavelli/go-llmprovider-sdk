#!/usr/bin/env python3
"""redgreen: named tests, each red or each green, at a revision, on a clone, never in the tree.

Defined by docs/decisions/0067-MADR-keep-four-more-scratch-helpers-as-tools.md
D1: a new test is seen to fail on the code before its fix, and the tests a
record names are seen to pass, none missing. `go test -run` passes a name
that matches nothing; this tool does not. The spec is JSON:

  {"tests": {"./pkg/": ["TestName", ...]},
   "files": ["pkg/x_test.go", ...],            (only with --rev)
   "drop": ["TestName", ...]}                  (only with --rev)

With --rev, the repository is cloned at REV, each of "files" is copied into
it from the working tree, and each function in "drop" is removed from those
files with the comment lines above it, so a test whose subject does not
exist at REV does not stop its package from building. Without --rev, the
working tree is cloned with tools/_common.scratch_clone.

Each package is run once, `go test -count=1 -v -run '^(names)$'`. With
--expect pass each name must print `--- PASS: name`; with --expect fail,
`--- FAIL: name`. A name with neither line, or in a package that does not
build, is a finding: a red run needs the test to run and fail on its
assertion (0067-MADR Q1). Each package's output is written to --out.

Usage:
  tools/redgreen.py [--root DIR] [--out DIR] SPEC.json --expect fail|pass [--rev REV]

Exit codes: 0 every name as expected, 1 a name not as expected or missing,
2 tool error (a bad spec, a drop that does not match once, a failed clone).
"""
from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

from _common import ROOT, out_dir, run, scratch_clone

NAME = re.compile(r"^(Test|Example)\w+$")
RESULT = re.compile(r"^\s*--- (PASS|FAIL): (\S+) ")
NOT_BUILT = ("[build failed]", "[setup failed]")
SPEC_KEYS = {"tests", "files", "drop"}


class SpecError(ValueError):
    """A spec or a drop the tool cannot use: exit 2."""


def load_spec(path: Path, rev: str | None) -> dict:
    """The spec at path, checked before anything runs."""
    spec = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(spec, dict) or set(spec) - SPEC_KEYS:
        raise SpecError(f"the spec's keys are {sorted(SPEC_KEYS)}")
    tests = spec.get("tests")
    if not isinstance(tests, dict) or not tests:
        raise SpecError('"tests" must map a package to its names')
    for pkg, names in tests.items():
        if not isinstance(names, list) or not names or not all(isinstance(n, str) and NAME.match(n) for n in names):
            raise SpecError(f"{pkg}: each name must match {NAME.pattern}")
    if rev is None and ("files" in spec or "drop" in spec):
        raise SpecError('"files" and "drop" need --rev')
    for name in spec.get("drop", []):
        if not NAME.match(name):
            raise SpecError(f"drop {name!r} must match {NAME.pattern}")
        if any(name in names for names in tests.values()):
            raise SpecError(f"{name} is both run and dropped")
    return spec


def drop_func(text: str, name: str) -> tuple[str, int]:
    """text without func name and the // lines directly above it, and how many were removed."""
    pattern = re.compile(r"(?:^//[^\n]*\n)*^func " + re.escape(name) + r"\(.*?^}\n", re.M | re.S)
    return pattern.subn("", text)


def prepare(root: Path, dest: Path, rev: str | None, files: list[str], drops: list[str]) -> None:
    """dest as the spec runs it: root at rev with files copied in and drops removed, or root's working tree."""
    if rev is None:
        scratch_clone(root, dest)
        return
    run(["git", "clone", "--quiet", "--shared", str(root), str(dest)], cwd=root, check=True)
    run(["git", "checkout", "--quiet", rev], cwd=dest, check=True)
    texts = {}
    for rel in files:
        source = root / rel
        if not source.is_file():
            raise SpecError(f"{rel} is not a file in the working tree")
        texts[rel] = source.read_text(encoding="utf-8")
    for name in drops:
        removed = 0
        for rel, text in texts.items():
            texts[rel], n = drop_func(text, name)
            removed += n
        if removed != 1:
            raise SpecError(f"drop {name}: {removed} function(s) removed, want 1")
    for rel, text in texts.items():
        (dest / rel).parent.mkdir(parents=True, exist_ok=True)
        (dest / rel).write_text(text, encoding="utf-8")


def results(output: str, names: list[str]) -> dict[str, str]:
    """Each name's result in a `go test -v` output: pass, fail, or missing."""
    seen: dict[str, str] = {}
    for line in output.splitlines():
        m = RESULT.match(line)
        if m and m.group(2) in names and seen.get(m.group(2)) != "fail":
            seen[m.group(2)] = m.group(1).lower()
    return {n: seen.get(n, "missing") for n in names}


def output_name(pkg: str) -> str:
    """The file a package's output goes to: ./llmprovider/catalog/ is llmprovider_catalog.out, . is root.out."""
    return (re.sub(r"[^\w-]+", "_", pkg).strip("_") or "root") + ".out"


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository (default: this one)")
    parser.add_argument("--out", type=Path, default=None, help="where each package's output goes")
    parser.add_argument("--rev", default=None, help="the revision to run at (default: the working tree)")
    parser.add_argument("--expect", choices=("fail", "pass"), required=True, help="what every name must do")
    parser.add_argument("spec", type=Path, help="the JSON spec")
    args = parser.parse_args(argv)
    root = args.root.resolve()
    out = args.out or out_dir("redgreen")
    out.mkdir(parents=True, exist_ok=True)

    work = Path(tempfile.mkdtemp(prefix="redgreen-"))
    clone = work / "clone"
    try:
        try:
            spec = load_spec(args.spec, args.rev)
            prepare(root, clone, args.rev, spec.get("files", []), spec.get("drop", []))
        except (OSError, ValueError) as err:
            print(f"redgreen: {err}", file=sys.stderr)
            return 2
        except subprocess.CalledProcessError as err:
            print(f"redgreen: {' '.join(err.cmd)}: {(err.stderr or '').strip()}", file=sys.stderr)
            return 2
        want = args.expect
        total, wrong = 0, []
        for pkg, names in spec["tests"].items():
            try:
                p = run(["go", "test", "-count=1", "-v", "-run", "^(" + "|".join(names) + ")$", pkg], cwd=clone)
            except OSError as err:
                print(f"redgreen: cannot run go test: {err}", file=sys.stderr)
                return 2
            output = p.stdout + p.stderr
            (out / output_name(pkg)).write_text(output, encoding="utf-8")
            built = not any(marker in output for marker in NOT_BUILT)
            print(f"== {pkg}: exit {p.returncode}" + ("" if built else ", did not build"))
            for name, got in results(output, names).items():
                total += 1
                label = got if built else "missing (did not build)"
                ok = built and got == want
                print(f"   {'ok ' if ok else 'BAD'} {label} {name}")
                if not ok:
                    wrong.append(name)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    print(f"redgreen: {total - len(wrong)} of {total} {want} as expected; output in {out}")
    return 1 if wrong else 0


if __name__ == "__main__":
    sys.exit(main())
