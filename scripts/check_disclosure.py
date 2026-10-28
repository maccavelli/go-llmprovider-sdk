#!/usr/bin/env python3
"""disclosure-check: no file names a real home directory.

Defined by docs/decisions/0063-MADR-align-docs-ci-and-disclosure-gates.md
R2 and R3. It reads every file git tracks or would add (or the files named)
and reports:
  * a macOS or Linux home path: /Users/<name>/ or /home/<name>/;
  * a Windows home path: a drive letter, then Users and a name, each after
    a slash or one or two backslashes, then a slash or a backslash;
unless the name is the placeholder <user>. It reads nothing of the host it
runs on: no account name, no host name and no deny list. Those stay with
tools/denyscan.py and the owner's pre-push guard, outside CI (AGENTS.md,
"Identifiers"). A hit is printed as path:line and its kind, never the path
it found. A file that cannot be read fails the check.

Usage:
  scripts/check_disclosure.py [--root DIR] [FILE...]

Exit codes: 0 clean, 1 a hit, 2 tool error (a file that cannot be read).
"""
from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SEP = r"(?:/|\\\\|\\)"
KINDS = (
    ("a macOS or Linux home path", re.compile(r"/(Users|home)/(?!<user>)[A-Za-z0-9._-]+/")),
    ("a Windows home path", re.compile(r"\b[A-Za-z]:" + SEP + r"Users" + SEP + r"(?!<user>)[A-Za-z0-9._ -]+(?:/|\\)",
                                       re.I)),
)


def tracked_files(root: Path) -> list[str]:
    """The files git tracks or would add, as paths relative to root."""
    out = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "-z"], cwd=root,
                         capture_output=True, check=True).stdout.decode()
    return [rel for rel in out.split("\0") if rel]


def scan_text(name: str, text: str) -> list[str]:
    """The hits in text, as "name:line: kind"."""
    return [f"{name}:{n}: {kind}"
            for n, line in enumerate(text.splitlines(), 1)
            for kind, rx in KINDS if rx.search(line)]


def scan(root: Path, names: list[str]) -> tuple[list[str], int]:
    """The hits in the named files that exist, and the number read. Bytes
    that are not UTF-8 are replaced, so a binary file is read as well."""
    hits: list[str] = []
    read = 0
    for name in names:
        path = root / name
        if not path.is_file():
            continue  # deleted, not yet staged
        hits.extend(scan_text(name, path.read_bytes().decode("utf-8", errors="replace")))
        read += 1
    return hits, read


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository (default: this one)")
    parser.add_argument("files", nargs="*", help="the files to read (default: every file git tracks or would add)")
    args = parser.parse_args(argv)
    root = args.root.resolve()
    try:
        hits, read = scan(root, args.files or tracked_files(root))
    except (OSError, subprocess.CalledProcessError) as err:
        print(f"disclosure-check: {err}", file=sys.stderr)
        return 2
    for h in hits:
        print(h)
    print(f"disclosure-check: {len(hits)} hit(s) in {read} file(s)")
    return 1 if hits else 0


if __name__ == "__main__":
    sys.exit(main())
