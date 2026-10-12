#!/usr/bin/env python3
"""plant: each plant in a spec must fail its tests, on a scratch clone, never in the tree.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1, for the
rule that a check is unused until it has been seen to fail. The spec is JSON:

  {"packages": ["./llmprovider/..."], "run": "_0034$|TestWireGoldens",
   "command": ["go", "test", ...],            (optional; replaces the go test)
   "plants": [{"name", "file", "old", "new"}, ...]}

The working tree is cloned once, with tools/_common.scratch_clone. Each plant
replaces "old", which must occur exactly once, runs the command, prints the
failure lines, and restores the file.

Usage:
  tools/plant.py [--root DIR] SPEC.json

Exit codes: 0 every plant was caught, 1 a plant was not caught, 2 tool error or
a plant whose "old" does not match once.
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

from _common import ROOT, scratch_clone

FAIL_LINE = re.compile(r"_test\.go:\d+|^\s*--- FAIL|^FAIL\s|llmtest\.go:\d+|wirecase\.go:\d+|want .* got|^(FAIL|ERROR): |Error")


def fail_lines(output: str, limit: int = 12) -> list[str]:
    return [line.strip()[:240] for line in output.splitlines() if FAIL_LINE.search(line)][:limit]


def default_command(spec: dict) -> list[str]:
    args = ["go", "test", "-count=1"]
    if spec.get("run"):
        args += ["-run", spec["run"]]
    return args + list(spec.get("packages", ["./..."]))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository (default: this one)")
    parser.add_argument("spec", type=Path, help="the JSON spec")
    args = parser.parse_args(argv)
    try:
        spec = json.loads(args.spec.read_text(encoding="utf-8"))
        plants = spec["plants"]
        cmd = spec.get("command") or default_command(spec)
    except (OSError, ValueError, KeyError) as err:
        print(f"plant: bad spec: {err}", file=sys.stderr)
        return 2

    work = Path(tempfile.mkdtemp(prefix="plant-"))
    clone = work / "clone"
    try:
        try:
            scratch_clone(args.root.resolve(), clone)
        except (OSError, subprocess.CalledProcessError) as err:
            print(f"plant: cannot make the scratch clone: {err}", file=sys.stderr)
            return 2
        mismatched, uncaught = [], []
        for p in plants:
            path = clone / p["file"]
            original = path.read_text(encoding="utf-8")
            if (n := original.count(p["old"])) != 1:
                print(f"== {p['name']}: \"old\" matches {n} time(s) in {p['file']}, want 1")
                mismatched.append(p["name"])
                continue
            path.write_text(original.replace(p["old"], p["new"], 1), encoding="utf-8")
            try:
                result = subprocess.run(cmd, cwd=clone, capture_output=True, text=True)
            finally:
                path.write_text(original, encoding="utf-8")
            if result.returncode == 0:
                print(f"== {p['name']}: NOT CAUGHT (exit 0)")
                uncaught.append(p["name"])
                continue
            print(f"== {p['name']}: caught (exit {result.returncode})")
            for line in fail_lines(result.stdout + result.stderr):
                print("   " + line)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    if mismatched:
        print(f"plant: {len(mismatched)} plant(s) did not match: {', '.join(mismatched)}")
        return 2
    if uncaught:
        print(f"plant: {len(uncaught)} plant(s) not caught: {', '.join(uncaught)}")
        return 1
    print(f"plant: all {len(plants)} plant(s) caught")
    return 0


if __name__ == "__main__":
    sys.exit(main())
