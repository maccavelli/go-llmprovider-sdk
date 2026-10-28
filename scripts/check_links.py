#!/usr/bin/env python3
"""links-check: every relative link in every Markdown file resolves.

Defined by docs/decisions/0063-MADR-align-docs-ci-and-disclosure-gates.md
R1 and R2, from tools/linkcheck.py (0035-MADR D1), which now imports it. It
reads every Markdown file git tracks or would add, so a link in a new record
is checked before it is staged. Code fences and inline code are skipped;
links with a scheme (https:, mailto:) and pure #anchors are not checked.

--lint-files prints, one per line, the files `make markdownlint` lints: the
same Markdown files, less the records under docs/decisions/ and
docs/reports/, which keep their own conventions (0047-MADR A2). The local
gate and both workflows lint this one list.

Usage:
  scripts/check_links.py [--root DIR]
  scripts/check_links.py --lint-files [--root DIR]

Exit codes: 0 clean, 1 a broken link, 2 tool error (or, with --lint-files,
no file, or a file name with white space, which the make target would split).
"""
from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
RECORD_DIRS = ("docs/decisions/", "docs/reports/")
LINK = re.compile(r"!?\[[^\]]*\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
SCHEME = re.compile(r"^[a-z][a-z0-9+.-]*:", re.I)
INLINE_CODE = re.compile(r"`[^`]*`")


def markdown_files(root: Path) -> list[str]:
    """The Markdown files git tracks or would add, as paths relative to root."""
    out = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "-z", "--", "*.md"], cwd=root,
                         capture_output=True, check=True).stdout.decode()
    return [rel for rel in out.split("\0") if rel]


def lint_files(root: Path) -> list[str]:
    """The Markdown files markdownlint lints: all but the records, sorted."""
    return sorted(f for f in markdown_files(root) if not f.startswith(RECORD_DIRS))


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
    parser.add_argument("--lint-files", action="store_true", help="print the files markdownlint lints instead")
    args = parser.parse_args(argv)
    root = args.root.resolve()
    try:
        if args.lint_files:
            files = lint_files(root)
            spaced = [f for f in files if any(c.isspace() for c in f)]
            if not files or spaced:
                print(f"links-check: {'no Markdown file to lint' if not files else f'names with white space: {spaced}'}",
                      file=sys.stderr)
                return 2
            print("\n".join(files))
            return 0
        problems, checked, files_read = check(root)
    except (OSError, UnicodeDecodeError, subprocess.CalledProcessError) as err:
        print(f"links-check: {err}", file=sys.stderr)
        return 2
    for p in problems:
        print(p)
    print(f"links-check: {len(problems)} problem(s), {checked} relative link(s) in {files_read} file(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
