#!/usr/bin/env python3
"""catalogcheck: at a refresh, report how many of each open catalog's six changed.

Defined by docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md
D7, and built in 0041-PLAN-decide-per-model-behaviour-from-metadata.md
Phase 7. It reports and never edits: in a live menu, a change caused by new
facts is the point.

Without --dir it records a new snapshot with
`go run ./llmprovider/internal/cmd/catalogs -snapshot DIR`, into --out, which
must not exist, or a new directory under the system's temporary directory.
The command reads each provider's key from that provider's variable. This
tool reads no key, prints none, and relays which listings the command
skipped. With --dir it checks an existing snapshot, and sends no request.

TestCatalogCheck0041_Sixes, run with LLMPROVIDER_SNAPSHOT_DIR=DIR, prints
each open catalog's six under each
profile, for DIR and for the baseline: --baseline, or the newest committed
llmprovider/catalog/testdata/ranking-*/ that holds a snapshot.json.

It also ran 0041-MADR D7's agreement tests, each rule beside the facts; they
went with the rules they compared, R1, R4, R5 and R6, which 0062-MADR D8
removed.

Usage:
  tools/catalogcheck.py [--dir DIR | --out DIR] [--baseline DIR] [--doc FILE]

Exit codes: 0 no difference, 1 a six changed, 2 tool error.
"""
from __future__ import annotations

import argparse
import re
import sys
import tempfile
from dataclasses import dataclass, field
from pathlib import Path

from _common import ROOT, out_dir, run

ENV = "LLMPROVIDER_SNAPSHOT_DIR"
SNAPSHOTS = ROOT / "llmprovider" / "catalog" / "testdata"
SIX = re.compile(r"^SIX\t(\S+)\t(\S+)\t(.*)$")
SKIPPED = re.compile(r"unset; its listing is skipped|: listing: ")


class ToolError(Exception):
    """A step that could not run: the tool's exit 2."""


@dataclass
class Report:
    """What one check found."""

    lines: list[str] = field(default_factory=list)
    findings: int = 0


def latest_snapshot() -> Path | None:
    """The newest committed ranking snapshot: testdata/ranking-YYYY-MM-DD with a snapshot.json."""
    dirs = sorted(d for d in SNAPSHOTS.glob("ranking-*") if (d / "snapshot.json").is_file())
    return dirs[-1] if dirs else None


def tail(text: str, n: int = 8) -> str:
    return "\n".join(text.strip().splitlines()[-n:])


def record(dest: Path, doc: Path | None) -> list[str]:
    """Record a snapshot into dest with the catalogs command; return the listings it skipped."""
    args = ["go", "run", "./llmprovider/internal/cmd/catalogs", "-snapshot", str(dest)]
    if doc is not None:
        args += ["-doc", str(doc)]
    res = run(args)
    if res.returncode != 0 or not (dest / "snapshot.json").is_file():
        raise ToolError(f"the catalogs command failed (exit {res.returncode}):\n{tail(res.stderr)}")
    return [line.strip() for line in res.stderr.splitlines() if SKIPPED.search(line)]


def sixes(snapshot: Path) -> dict[tuple[str, str], list[str] | None]:
    """Each open catalog's six under each profile, from snapshot; None for a missing listing."""
    res = run(["go", "test", "-count=1", "-v", "-run", "^TestCatalogCheck0041_Sixes$", "./llmprovider/catalog/"],
              env={ENV: str(snapshot)})
    if res.returncode != 0:
        raise ToolError(f"the sixes could not be read from {snapshot} (exit {res.returncode}):\n"
                        f"{tail(res.stdout + res.stderr)}")
    got: dict[tuple[str, str], list[str] | None] = {}
    for line in res.stdout.splitlines():
        if m := SIX.match(line):
            provider, profile, ids = m.groups()
            got[(provider, profile)] = None if ids == "missing" else [i for i in ids.split(",") if i]
    if not got:
        raise ToolError(f"TestCatalogCheck0041_Sixes printed nothing for {snapshot}")
    return got


def compare(baseline: dict[tuple[str, str], list[str] | None],
            new: dict[tuple[str, str], list[str] | None], name: str) -> Report:
    """How many of each six changed since the baseline; a listing either side lacks is noted, not counted."""
    report = Report()
    lines = []
    providers = sorted({p for p, _ in baseline} | {p for p, _ in new})
    for provider in providers:
        if new.get((provider, "-"), "") is None or baseline.get((provider, "-"), "") is None:
            side = "this snapshot" if new.get((provider, "-"), "") is None else "the baseline"
            lines.append(f"  {provider}: not compared; {side} has no listing")
            continue
        for profile in ("utility", "capable", "agent"):
            old, now = baseline.get((provider, profile)), new.get((provider, profile))
            if old is None or now is None:
                continue
            added = [i for i in now if i not in old]
            removed = [i for i in old if i not in now]
            if added:
                report.findings += 1
                lines.append(f"  {provider} {profile}: {len(added)} of {len(now)} changed: "
                             f"+{', +'.join(added)}; -{', -'.join(removed)}")
            else:
                lines.append(f"  {provider} {profile}: unchanged")
    report.lines = [f"sixes against {name}: {report.findings} changed"] + lines
    return report


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    where = parser.add_mutually_exclusive_group()
    where.add_argument("--dir", type=Path, help="check this existing snapshot; send no request")
    where.add_argument("--out", type=Path, help="record the new snapshot here; it must not exist")
    parser.add_argument("--baseline", type=Path, help="compare the sixes with this snapshot (default: the newest committed)")
    parser.add_argument("--doc", type=Path, help="pass this models.dev-format document to the command")
    args = parser.parse_args(argv)
    try:
        baseline = args.baseline or latest_snapshot()
        if baseline is None or not (baseline / "snapshot.json").is_file():
            raise ToolError(f"no baseline snapshot: {baseline}")
        if args.dir is not None:
            snapshot = args.dir.resolve()
            if not (snapshot / "snapshot.json").is_file():
                raise ToolError(f"{snapshot}: no snapshot.json")
            skipped: list[str] = []
        else:
            snapshot = (args.out or Path(tempfile.mkdtemp(dir=out_dir("catalogcheck"))) / "snapshot").resolve()
            skipped = record(snapshot, args.doc)
        print(f"catalogcheck: snapshot {snapshot}")
        for line in skipped:
            print(f"skipped: {line}")
        name = baseline.relative_to(ROOT) if baseline.is_relative_to(ROOT) else baseline
        changed = compare(sixes(baseline.resolve()), sixes(snapshot), str(name))
    except ToolError as err:
        print(f"catalogcheck: {err}", file=sys.stderr)
        return 2
    for line in changed.lines:
        print(line)
    findings = changed.findings
    print(f"catalogcheck: {findings} finding(s); nothing was changed")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
