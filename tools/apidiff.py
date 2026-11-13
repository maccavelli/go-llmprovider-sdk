#!/usr/bin/env python3
"""apidiff: every exported API change between two refs, or a ref and the working tree.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1. Where
`make api-check` fails only on an incompatible change against the latest v1
tag, this prints every change, compatible or not, between any two points, as a
record's release notes need. apidiff runs at the version scripts/check_api.py
pins, read from that file, so there is one pin, and once for each of its
SUBTREES, the reports merged.

Usage:
  tools/apidiff.py OLD [NEW]

OLD and NEW are git refs; without NEW, the working tree is compared.

Exit codes: 0 no incompatible change, 1 an incompatible change, 2 tool error.
"""
from __future__ import annotations

import argparse
import io
import re
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path

from _common import ROOT

MODULE = "github.com/maccavelli/go-llmprovider-sdk"
PIN = re.compile(r'^APIDIFF = "([^"]+)"$', re.M)
SUBTREES = re.compile(r"^SUBTREES = \(([^)]*)\)$", re.M)


def pinned() -> str:
    """The apidiff module and version scripts/check_api.py runs."""
    match = PIN.search((ROOT / "scripts" / "check_api.py").read_text(encoding="utf-8"))
    if not match:
        raise ValueError("scripts/check_api.py pins no APIDIFF")
    return match.group(1)


def subtrees() -> list[str]:
    """The subtrees scripts/check_api.py compares, one apidiff run each: the
    pattern `<module>/...` also matches the nested module authcontract, and
    apidiff then files the whole export under its path (0074-PLAN Phase 4)."""
    match = SUBTREES.search((ROOT / "scripts" / "check_api.py").read_text(encoding="utf-8"))
    if not match:
        raise ValueError("scripts/check_api.py names no SUBTREES")
    names = re.findall(r'"([^"]+)"', match.group(1))
    if not names:
        raise ValueError("scripts/check_api.py's SUBTREES is empty")
    return names


def merge(reports: list[str]) -> str:
    """One report from the per-subtree ones: each section's lines together, the
    incompatible changes first, in apidiff's own layout."""
    sections: dict[str, list[str]] = {"Incompatible changes:": [], "Compatible changes:": []}
    current = ""
    for report in reports:
        for line in report.splitlines():
            if line in sections:
                current = line
            elif line.startswith("- ") and current:
                sections[current].append(line)
    out = []
    for header, lines in sections.items():
        if lines:
            out.append(header)
            out.extend(lines)
    return "\n".join(out) + "\n" if out else ""


def archive(ref: str, dest: Path) -> None:
    data = subprocess.run(["git", "archive", ref], cwd=ROOT, capture_output=True, check=True).stdout
    with tarfile.open(fileobj=io.BytesIO(data)) as tar:
        tar.extractall(dest, filter="data")


def go_run(tool: str, args: list[str], cwd: Path) -> str:
    return subprocess.run(["go", "run", tool, *args], cwd=cwd, capture_output=True, text=True, check=True).stdout


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("old", help="the older git ref")
    parser.add_argument("new", nargs="?", help="the newer git ref (default: the working tree)")
    args = parser.parse_args(argv)
    try:
        tool = pinned()
        with tempfile.TemporaryDirectory() as tmp:
            work = Path(tmp)
            archive(args.old, work / "old")
            new_tree = ROOT
            if args.new:
                new_tree = work / "new"
                archive(args.new, new_tree)
            reports = []
            for subtree in subtrees():
                old, new = work / f"old-{subtree}.x", work / f"new-{subtree}.x"
                go_run(tool, ["-m", "-w", str(old), f"{MODULE}/{subtree}"], work / "old")
                go_run(tool, ["-m", "-w", str(new), f"{MODULE}/{subtree}"], new_tree)
                reports.append(go_run(tool, ["-m", str(old), str(new)], ROOT))
            report = merge(reports)
    except (OSError, ValueError, subprocess.CalledProcessError) as err:
        detail = getattr(err, "stderr", "") or ""
        print(f"apidiff: {err} {detail}".strip(), file=sys.stderr)
        return 2
    print(report, end="")
    return 1 if "Incompatible changes" in report else 0


if __name__ == "__main__":
    sys.exit(main())
