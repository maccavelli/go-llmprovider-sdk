#!/usr/bin/env python3
"""gate: every local check, on a scratch clone of the working tree, staged there.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1. It runs
0026-PLAN's 22 checks, with the link check and markdownlint by CI's make
targets and disclosure-check added (0063-MADR R1, R2), then tools-test
(0035-MADR Q1), on a shared clone of
the repository with the working tree's changes copied in and committed with
`git commit-tree`. The checks therefore see exactly what would be staged, and
the repository's own index is never touched.

It starts only once the one-minute load average is below --max-load, the CPU
count by default: some tests time out on a loaded host (0034-PLAN Phase 4e,
amended).

With --markdown it runs only what a Markdown-only change needs (0067-MADR
D4): the checks whose names are scripts/test_gates.py's MARKDOWN_GATES, which
CI's docs workflow runs, the identifier scan, and, in --src, which they only
read, tools/recordlint.py --changed and tools/claims.py on the changed records
that hold an evidence index (none is clean). It waits for the load as the full gate does, since
gate-selftest runs Go tests (0067-MADR A2), and refuses a change to any file
that is not Markdown, deletions included: such a change needs the full gate.

Each check's full output is written to OUT/<name>.out, and the exit codes to
OUT/summary.out.

Usage:
  tools/gate.py [--src DIR] [--out DIR] [--copy DIR] [--max-load N] [--timeout SECONDS]
  tools/gate.py --markdown [--src DIR] [--out DIR] [--copy DIR]

Exit codes: 0 every check passed, 1 a check failed or --markdown refused the
change, 2 tool error or the load never fell.
"""
from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path

import claims
import denyscan
from _common import ROOT, load_limit, out_dir, scratch_clone, wait_load
from test_gates import MARKDOWN_GATES

# A record's path, as scripts/check_records.py names records.
RECORD_PATH = re.compile(r"^docs/(decisions|reports)/\d{4}-(MADR|PLAN|REPORT|GATES)-[a-z0-9-]+\.md$")


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
    # Each test twice in one process: one that reads what its own first run
    # left in a process-wide cache fails (0058-MADR's amendment of 2026-10-24).
    Check("test-repeat", command(["go", "test", "-count=2", "./..."])),
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
    make("changelog-check"),
    make("docs-check"),
    make("gate-selftest"),
    # CI's documentation gates, by their make targets, so the local gate and
    # both workflows lint and check one file list (0063-MADR R1, R2); the
    # local markdownlint-cli2 stands in for CI's npx at the same pin.
    Check("markdownlint", command(["make", "markdownlint", "MARKDOWNLINT=markdownlint-cli2"])),
    Check("g-wire-stable", command(["go", "test", "-count=3", "./llmprovider/...", "-run", "TestWireGoldens"])),
    make("links-check"),
    make("disclosure-check"),
    Check("deny-scan", deny),
    make("tools-test"),
]


def in_tree(src: Path, args: list[str]) -> Callable[[Path, list[str]], tuple[int, str]]:
    """A check run in src, not the copy: it compares the tree with HEAD, and the copy's HEAD holds the change."""
    def run(_copy: Path, changed: list[str]) -> tuple[int, str]:
        return command(args)(src, changed)
    return run


def claims_on_records(src: Path) -> Callable[[Path, list[str]], tuple[int, str]]:
    """tools/claims.py, in src, on the records the change touches that hold an evidence index, by claims.py's own
    MARKER; clean when there are none, where claims.py --changed calls none a tool error and a record with no
    rows a finding (0067-MADR A2, A3)."""
    def run(_copy: Path, changed: list[str]) -> tuple[int, str]:
        records = [p for p in changed
                   if RECORD_PATH.match(p) and claims.MARKER.search((src / p).read_text(encoding="utf-8"))]
        if not records:
            return 0, "claims: no record with an evidence index changed\n"
        return command(["python3", "-B", "tools/claims.py", *records])(src, changed)
    return run


def markdown_checks(src: Path) -> list[Check]:
    """--markdown's checks: the gate's own whose names CI's docs workflow runs (MARKDOWN_GATES), in its order,
    then the identifier scan, and, in src, the record lint and the claims (0067-MADR D4, A2)."""
    by_name = {c.name: c for c in CHECKS}
    missing = [name for name in MARKDOWN_GATES if name not in by_name]
    if missing:
        raise ValueError(f"MARKDOWN_GATES names no gate check: {', '.join(missing)}")
    return [*(by_name[name] for name in MARKDOWN_GATES), by_name["deny-scan"],
            Check("recordlint", in_tree(src, ["python3", "-B", "tools/recordlint.py", "--changed"])),
            Check("claims", claims_on_records(src))]


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--src", type=Path, default=ROOT, help="the repository to gate (default: this one)")
    parser.add_argument("--out", type=Path, default=None, help="where each check's output goes")
    parser.add_argument("--copy", type=Path, default=None, help="where the scratch clone goes")
    parser.add_argument("--max-load", type=float, default=load_limit(), help="start below this load (default: CPU count)")
    parser.add_argument("--timeout", type=float, default=25 * 60, help="seconds to wait for the load to fall")
    parser.add_argument("--markdown", action="store_true",
                        help="only the checks a Markdown-only change needs (0067-MADR D4)")
    args = parser.parse_args(argv)
    src = args.src.resolve()
    out = args.out or out_dir("gate")
    copy = args.copy or out_dir("gate-copy")
    out.mkdir(parents=True, exist_ok=True)

    # --markdown waits too: its gate-selftest runs Go tests (0067-MADR A2).
    checks = markdown_checks(src) if args.markdown else CHECKS
    print(f"load {os.getloadavg()[0]:.1f}; waiting for it to fall below {args.max_load:g}", flush=True)
    if not wait_load(args.max_load, args.timeout):
        print(f"gate: the load did not fall below {args.max_load:g} within {args.timeout:g} s; nothing run")
        return 2
    print(f"started at load {os.getloadavg()[0]:.1f}", flush=True)
    try:
        changed = scratch_clone(src, copy)
        # Every path the change touches, deletions included, which scratch_clone's list leaves out.
        touched = subprocess.run(["git", "diff", "--name-only", "HEAD^", "HEAD"], cwd=copy, capture_output=True,
                                 text=True, check=True).stdout.split() if args.markdown else []
    except (OSError, subprocess.CalledProcessError) as err:
        print(f"gate: cannot make the scratch clone: {err}", file=sys.stderr)
        return 2
    if code := [p for p in touched if not p.endswith(".md")]:
        for path in code:
            print(f"gate: {path} is not Markdown; a change to it needs the full gate")
        print("gate --markdown: nothing run")
        return 1

    summary = []
    for check in checks:
        rc, text = check.run(copy, changed)
        (out / f"{check.name}.out").write_text(text, encoding="utf-8")
        summary.append(f"{check.name}: rc={rc}")
        print(summary[-1], flush=True)
    (out / "summary.out").write_text("\n".join(summary) + "\n", encoding="utf-8")
    failed = [s for s in summary if not s.endswith("rc=0")]
    print(f"gate: {len(checks) - len(failed)} of {len(checks)} passed; output in {out}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
