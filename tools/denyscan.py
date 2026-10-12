#!/usr/bin/env python3
"""denyscan: no file names this machine's account, its host, or a real home path.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1, for
AGENTS.md's "Identifiers" rule. The account name, the host name and its
domain are read at run time, so this file holds none of them. It prints the
file, the line and the kind of hit, never the identifier itself. A
placeholder home path, such as /home/<user>/, is not a hit.

Usage:
  tools/denyscan.py [--root DIR] FILE...
  tools/denyscan.py [--root DIR] --changed

--changed scans the files `git status` lists as added, changed or untracked.
A relative FILE is read from the root.

Exit codes: 0 clean, 1 a hit, 2 tool error.
"""
from __future__ import annotations

import argparse
import getpass
import re
import socket
import subprocess
import sys
from pathlib import Path

from _common import ROOT

HOME_PATH = re.compile(r"/(Users|home)/(?!<user>)[A-Za-z0-9._-]+/")


def needles() -> dict[str, re.Pattern[str]]:
    """The identifiers to look for, by kind, each matched as a whole word."""
    host = socket.gethostname().split(".")[0]
    fqdn = socket.getfqdn()
    domain = fqdn.split(".", 1)[1] if "." in fqdn else ""
    found = {"the local account name": getpass.getuser(), "the hostname": host}
    if domain not in ("", "local", "localdomain"):
        found["the hostname domain"] = domain
    return {kind: re.compile(r"(?<![A-Za-z0-9])" + re.escape(value) + r"(?![A-Za-z0-9])", re.I)
            for kind, value in found.items() if len(value) >= 3}


def changed_files(root: Path) -> list[str]:
    """The files git lists as added, modified, renamed or untracked, deletions excluded."""
    out = subprocess.run(["git", "status", "--porcelain", "-z", "--untracked-files=all"], cwd=root,
                         capture_output=True, check=True).stdout.decode()
    entries = out.split("\0")
    files, i = [], 0
    while i < len(entries):
        entry = entries[i]
        i += 1
        if not entry:
            continue
        status, path = entry[:2], entry[3:]
        if "R" in status or "C" in status:
            i += 1  # the rename's source follows
        if "D" not in status:
            files.append(path)
    return files


def scan(root: Path, names: list[str]) -> tuple[list[str], int]:
    """The hits, as "file:line: kind", and the number of files read."""
    patterns = needles()
    hits: list[str] = []
    read = 0
    for name in names:
        path = Path(name) if Path(name).is_absolute() else root / name
        if not path.is_file():
            continue
        read += 1
        for n, line in enumerate(path.read_text(encoding="utf-8", errors="replace").splitlines(), 1):
            hits.extend(f"{name}:{n}: {kind}" for kind, rx in patterns.items() if rx.search(line))
            if HOME_PATH.search(line):
                hits.append(f"{name}:{n}: a machine home path")
    return hits, read


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository (default: this one)")
    parser.add_argument("--changed", action="store_true", help="scan the files git lists as changed")
    parser.add_argument("files", nargs="*", help="the files to scan")
    args = parser.parse_args(argv)
    root = args.root.resolve()
    try:
        names = changed_files(root) if args.changed else args.files
        hits, read = scan(root, names)
    except (OSError, subprocess.CalledProcessError) as err:
        print(f"denyscan: {err}", file=sys.stderr)
        return 2
    for h in hits:
        print(h)
    print(f"deny-scan: {len(hits)} hit(s) in {read} file(s)")
    return 1 if hits else 0


if __name__ == "__main__":
    sys.exit(main())
