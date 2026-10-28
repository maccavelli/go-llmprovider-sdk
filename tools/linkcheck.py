#!/usr/bin/env python3
"""linkcheck: every relative link in every tracked or untracked Markdown file resolves.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1. The
check is CI's links-check, scripts/check_links.py, which this tool imports
(0063-MADR R2); this keeps the local command and its --detail file.
Untracked files that are not ignored count, so a link in a new record is
checked before it is staged. Code fences and inline code are skipped; links
with a scheme (https:, mailto:) and pure #anchors are not checked.

Usage:
  tools/linkcheck.py [--root DIR] [--detail FILE]

Exit codes: 0 clean, 1 a broken link, 2 tool error.
"""
from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

from _common import ROOT
from check_links import check


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository to check (default: this one)")
    parser.add_argument("--detail", type=Path, help="also write the broken links to this file")
    args = parser.parse_args(argv)
    try:
        problems, checked, files = check(args.root.resolve())
    except (OSError, UnicodeDecodeError, subprocess.CalledProcessError) as err:
        print(f"linkcheck: {err}", file=sys.stderr)
        return 2
    if args.detail:
        args.detail.write_text("".join(p + "\n" for p in problems), encoding="utf-8")
    for p in problems:
        print(p)
    print(f"G-links: {len(problems)} problem(s), {checked} relative link(s) in {files} file(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
