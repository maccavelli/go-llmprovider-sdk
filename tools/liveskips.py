#!/usr/bin/env python3
"""liveskips: sort a `go test -v` run into passes, skips with their reasons, and failures.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1, for the
live suites' record at each release. A test's reason is the last line it
logged before it ended. A test whose name holds a "/" is a subtest, even when
`go test` prints it unindented.

Usage:
  tools/liveskips.py RUN.txt

Exit codes: 0 no failure, 1 a test or a package failed, 2 tool error.
"""
from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

RUN = re.compile(r"^=== (?:RUN|CONT|PAUSE|NAME)\s+(\S+)")
LOG = re.compile(r"^\s+\S+\.go:\d+: (.*)")
END = re.compile(r"^(\s*)--- (PASS|SKIP|FAIL): (\S+) \(([^)]*)\)")
PACKAGE_FAIL = re.compile(r"^FAIL\s+\S+")


@dataclass
class Tally:
    """Passes, skips and failures, each skip and failure with its reason."""

    counts: dict[str, dict[str, int]] = field(
        default_factory=lambda: {k: {"PASS": 0, "SKIP": 0, "FAIL": 0} for k in ("top", "sub")})
    reasons: list[tuple[str, str, str]] = field(default_factory=list)
    package_failures: list[str] = field(default_factory=list)


def classify(lines: list[str]) -> Tally:
    tally, last, current = Tally(), {}, None
    for line in lines:
        if m := RUN.match(line):
            current = m.group(1)
        elif (m := LOG.match(line)) and current:
            last[current] = m.group(1)
        elif m := END.match(line):
            indent, status, name, _ = m.groups()
            kind = "sub" if indent or "/" in name else "top"
            tally.counts[kind][status] += 1
            if status != "PASS":
                tally.reasons.append((status, name, last.get(name, "")[:200]))
        elif PACKAGE_FAIL.match(line):
            tally.package_failures.append(line.strip())
    return tally


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("run", type=Path, help="a `go test -v` output")
    args = parser.parse_args(argv)
    try:
        lines = args.run.read_text(encoding="utf-8", errors="replace").splitlines()
    except OSError as err:
        print(f"liveskips: {err}", file=sys.stderr)
        return 2
    tally = classify(lines)
    for kind, label in (("top", "top level"), ("sub", "subtests")):
        c = tally.counts[kind]
        print(f"{label}: {c['PASS']} passed, {c['SKIP']} skipped, {c['FAIL']} failed")
    for status, name, reason in tally.reasons:
        print(f"{status} {name}: {reason}")
    for failure in tally.package_failures:
        print(f"package {failure}")
    failed = tally.counts["top"]["FAIL"] + tally.counts["sub"]["FAIL"] + len(tally.package_failures)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
