#!/usr/bin/env python3
"""linkcheck: every relative link in every tracked or untracked Markdown file resolves.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1; it is
the gate's g-links check. Untracked files that are not ignored count, so a
link in a new record is checked before it is staged. Code fences and inline
code are skipped; links with a scheme (https:, mailto:) and pure #anchors are
not checked.

Usage:
  tools/linkcheck.py [--root DIR] [--detail FILE]

Exit codes: 0 clean, 1 a broken link, 2 tool error.
"""
from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

from _common import ROOT

LINK = re.compile(r"!?\[[^\]]*\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
SCHEME = re.compile(r"^[a-z][a-z0-9+.-]*:", re.I)
INLINE_CODE = re.compile(r"`[^`]*`")


def markdown_files(root: Path) -> list[str]:
    out = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "-z", "--", "*.md"], cwd=root,
                         capture_output=True, check=True).stdout.decode()
    return [rel for rel in out.split("\0") if rel]


def check(root: Path) -> tuple[list[str], int, int]:
    """The broken links, as "file:line: target", the links checked, and the files read."""
    problems: list[str] = []
    checked = files = 0
    for rel in markdown_files(root):
        path = root / rel
        if not path.exists():
            continue
        files += 1
        fence = False
        for n, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            if line.lstrip().startswith(("```", "~~~")):
                fence = not fence
                continue
            if fence:
                continue
            for target in LINK.findall(INLINE_CODE.sub("", line)):
                if SCHEME.match(target) or target.startswith("#"):
                    continue
                checked += 1
                if not (path.parent / target.split("#", 1)[0]).resolve().exists():
                    problems.append(f"{rel}:{n}: {target}")
    return problems, checked, files


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository to check (default: this one)")
    parser.add_argument("--detail", type=Path, help="also write the broken links to this file")
    args = parser.parse_args(argv)
    try:
        problems, checked, files = check(args.root.resolve())
    except (OSError, subprocess.CalledProcessError) as err:
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
