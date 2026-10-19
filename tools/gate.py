#!/usr/bin/env python3
"""gate: every local check, on a scratch clone of the working tree, staged there.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1. It runs
0026-PLAN's 22 checks, then tools-test (0035-MADR Q1), on a shared clone of
the repository with the working tree's changes copied in and committed with
`git commit-tree`. The checks therefore see exactly what would be staged, and
the repository's own index is never touched.

It starts only once the one-minute load average is below --max-load, the CPU
count by default: some tests time out on a loaded host (0034-PLAN Phase 4e,
amended).

Each check's full output is written to OUT/<name>.out, and the exit codes to
OUT/summary.out.

Usage:
  tools/gate.py [--src DIR] [--out DIR] [--copy DIR] [--max-load N] [--timeout SECONDS]

Exit codes: 0 every check passed, 1 a check failed, 2 tool error or the load
never fell.
"""
from __future__ import annotations

import argparse
import os
import subprocess
import sys
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path

import denyscan
import linkcheck
from _common import ROOT, load_limit, out_dir, scratch_clone, wait_load

RECORD_DIRS = ("docs/decisions/", "docs/reports/")


@dataclass(frozen=True)
class Check:
    """One check: its name, and how to run it on the copy, given the changed files."""

    name: str
    run: Callable[[Path, list[str]], tuple[int, str]]


def command(args: list[str], env: dict[str, str] | None = None) -> Callable[[Path, list[str]], tuple[int, str]]:
    def run(copy: Path, _changed: list[str]) -> tuple[int, str]:
        try:
            p = subprocess.run(args, cwd=copy, capture_output=True, text=True, env={**os.environ, **(env or {})})
        except FileNotFoundError as err:
            return 127, str(err)
        return p.returncode, p.stdout + p.stderr
    return run


def markdown_files(root: Path) -> list[str]:
    """The Markdown files markdownlint checks: all but the records, which its config excludes."""
    return sorted(f for f in linkcheck.markdown_files(root) if not f.startswith(RECORD_DIRS))


def markdownlint(copy: Path, _changed: list[str]) -> tuple[int, str]:
    return command(["markdownlint-cli2", "--no-globs", *markdown_files(copy)])(copy, _changed)


def links(copy: Path, _changed: list[str]) -> tuple[int, str]:
    problems, checked, files = linkcheck.check(copy)
    text = "".join(p + "\n" for p in problems)
    return (1 if problems else 0), text + f"G-links: {len(problems)} problem(s), {checked} relative link(s) in {files} file(s)\n"


def deny(copy: Path, changed: list[str]) -> tuple[int, str]:
    hits, read = denyscan.scan(copy, changed)
    return (1 if hits else 0), "".join(h + "\n" for h in hits) + f"deny-scan: {len(hits)} hit(s) in {read} file(s)\n"


def make(target: str) -> Check:
    return Check(target, command(["make", target]))


VETS = [Check(f"vet-{goos}{suffix}", command(["go", "vet", *tags, "./..."], {"GOOS": goos, "CGO_ENABLED": "0"}))
        for goos in ("darwin", "linux", "windows")
        for suffix, tags in (("", []), ("-live", ["-tags", "live_gateways"]))]

CHECKS: list[Check] = [
    make("pre-add-check"),
    *VETS,
    Check("test-race-cover", command(["go", "test", "-race", "-count=1", "-cover", "./..."])),
    Check("test-shuffle", command(["go", "test", "-shuffle=on", "-count=1", "./..."])),
    Check("tidy-diff", command(["go", "mod", "tidy", "-diff"])),
    make("lint"),
    make("parity-check"),
    make("dep-check"),
    make("coverage-check"),
    make("api-check"),
    make("generate-check"),
    make("records-check"),
    make("tables-check"),
    make("fixtures-check"),
    make("gate-selftest"),
    Check("markdownlint", markdownlint),
    Check("g-wire-stable", command(["go", "test", "-count=3", "./llmprovider/...", "-run", "TestWireGoldens"])),
    Check("g-links", links),
    Check("deny-scan", deny),
    make("tools-test"),
]


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--src", type=Path, default=ROOT, help="the repository to gate (default: this one)")
    parser.add_argument("--out", type=Path, default=None, help="where each check's output goes")
    parser.add_argument("--copy", type=Path, default=None, help="where the scratch clone goes")
    parser.add_argument("--max-load", type=float, default=load_limit(), help="start below this load (default: CPU count)")
    parser.add_argument("--timeout", type=float, default=25 * 60, help="seconds to wait for the load to fall")
    args = parser.parse_args(argv)
    out = args.out or out_dir("gate")
    copy = args.copy or out_dir("gate-copy")
    out.mkdir(parents=True, exist_ok=True)

    print(f"load {os.getloadavg()[0]:.1f}; waiting for it to fall below {args.max_load:g}", flush=True)
    if not wait_load(args.max_load, args.timeout):
        print(f"gate: the load did not fall below {args.max_load:g} within {args.timeout:g} s; nothing run")
        return 2
    print(f"started at load {os.getloadavg()[0]:.1f}", flush=True)
    try:
        changed = scratch_clone(args.src.resolve(), copy)
    except (OSError, subprocess.CalledProcessError) as err:
        print(f"gate: cannot make the scratch clone: {err}", file=sys.stderr)
        return 2

    summary = []
    for check in CHECKS:
        rc, text = check.run(copy, changed)
        (out / f"{check.name}.out").write_text(text, encoding="utf-8")
        summary.append(f"{check.name}: rc={rc}")
        print(summary[-1], flush=True)
    (out / "summary.out").write_text("\n".join(summary) + "\n", encoding="utf-8")
    failed = [s for s in summary if not s.endswith("rc=0")]
    print(f"gate: {len(CHECKS) - len(failed)} of {len(CHECKS)} passed; output in {out}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
