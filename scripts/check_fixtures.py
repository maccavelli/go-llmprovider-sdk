#!/usr/bin/env python3
"""fixtures-check: recorded fixtures hold no account data.

Defined by docs/decisions/0047-MADR-keep-five-more-scratch-helpers-as-tools.md
D1; written in 0047-PLAN Phase 1. It scans every file under a testdata
directory that git tracks or would add (or the files named) for:

  * an e-mail address, unless at example.com, example.org or example.net, or
    under .test, .invalid or .example;
  * a UUID, unless the nil UUID;
  * an account, user or organization id: user-, org-, acct- or account-
    (or _), then eight or more letters and digits, at least one a digit or
    a capital, unless a placeholder (REDACTED, or a run of X);
  * an Anthropic workspace id: wrkspc_, then letters and digits, unless a
    placeholder (0064-MADR, amendment 2026-10-27);
  * a JWT: eyJ, then two dot-separated base64url parts;
  * a home path: /Users/<name> or /home/<name>.

A hit is printed as path:line:column, its kind, and a fingerprint: the first
12 hex digits of the value's SHA-256. The value itself, and any text around
it, is never printed.

scripts/fixtures-baseline.txt lists the hits a review admitted, one
"path kind fingerprint" per line; "#" starts a comment. A hit not in it is a
finding, and so is a line of it that no hit matches.

Usage:
  scripts/check_fixtures.py [FILE...]
  scripts/check_fixtures.py --write-baseline

--write-baseline writes the current hits as the baseline, keeping its
comments, and prints only their count. It is a person's step, after a review
of the hits.

Exit codes: 0 clean, 1 a finding, 2 tool error.
"""
from __future__ import annotations

import argparse
import hashlib
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
BASELINE = "scripts/fixtures-baseline.txt"
NIL_UUID = "00000000-0000-0000-0000-000000000000"
EXAMPLE_DOMAINS = ("example.com", "example.org", "example.net")
EXAMPLE_TLDS = (".test", ".invalid", ".example")
PLACEHOLDER = re.compile(r"^(?:REDACTED|X{4,})$", re.I)

EMAIL = re.compile(r"[A-Za-z0-9._%+-]+@([A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+)")
UUID = re.compile(r"\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b", re.I)
ACCOUNT = re.compile(r"\b(?:user|org|acct|account)[-_]([A-Za-z0-9]{8,})", re.I)
WORKSPACE = re.compile(r"\bwrkspc_([A-Za-z0-9]+)")
JWT = re.compile(r"\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}")
HOME = re.compile(r"/(?:Users|home)/[A-Za-z0-9._-]+")


@dataclass(frozen=True)
class Hit:
    """One account-data-shaped string: where it is, what kind, its fingerprint."""

    path: str
    line: int
    column: int
    kind: str
    fingerprint: str

    @property
    def key(self) -> str:
        """The hit's baseline line."""
        return f"{self.path} {self.kind} {self.fingerprint}"

    def __str__(self) -> str:
        return f"{self.path}:{self.line}:{self.column} {self.kind} {self.fingerprint}"


def fingerprint(value: str) -> str:
    """The first 12 hex digits of value's SHA-256."""
    return hashlib.sha256(value.encode("utf-8")).hexdigest()[:12]


def allowed_email(match: re.Match[str]) -> bool:
    domain = match.group(1).lower()
    return domain in EXAMPLE_DOMAINS or domain.endswith(EXAMPLE_TLDS)


def allowed_account(match: re.Match[str]) -> bool:
    tail = match.group(1)
    return PLACEHOLDER.match(tail) is not None or re.search(r"[0-9A-Z]", tail) is None


KINDS: list[tuple[str, re.Pattern[str], object]] = [
    ("email", EMAIL, allowed_email),
    ("uuid", UUID, lambda m: m.group(0) == NIL_UUID),
    ("account-id", ACCOUNT, allowed_account),
    ("workspace-id", WORKSPACE, lambda m: PLACEHOLDER.match(m.group(1)) is not None),
    ("jwt", JWT, lambda m: False),
    ("home-path", HOME, lambda m: False),
]


def scan_text(path: str, text: str) -> list[Hit]:
    """The hits in text, in order of line and column."""
    hits = []
    for number, line in enumerate(text.splitlines(), start=1):
        for kind, pattern, allowed in KINDS:
            for m in pattern.finditer(line):
                if not allowed(m):
                    hits.append(Hit(path, number, m.start() + 1, kind, fingerprint(m.group(0))))
    return sorted(hits, key=lambda h: (h.line, h.column, h.kind))


def fixture_files(root: Path) -> list[str]:
    """The files under a testdata directory that git tracks or would add."""
    out = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "-z"], cwd=root,
                         capture_output=True, check=True).stdout.decode()
    return sorted(f for f in filter(None, out.split("\0"))
                  if "testdata" in Path(f).parts and (root / f).is_file())


def scan(root: Path, files: list[str]) -> list[Hit]:
    """The hits in files, read from root; a file that is not UTF-8 text is skipped."""
    hits = []
    for f in files:
        try:
            text = (root / f).read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue
        hits.extend(scan_text(f, text))
    return hits


def read_baseline(path: Path) -> tuple[list[str], set[str]]:
    """The baseline's comment lines, and its entries."""
    if not path.exists():
        return [], set()
    comments, entries = [], set()
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith("#") or not line.strip():
            comments.append(line)
        elif len(line.split()) == 3:
            entries.add(line.strip())
        else:
            raise ValueError(f"{BASELINE}: not 'path kind fingerprint': {line!r}")
    return comments, entries


def findings(hits: list[Hit], baseline: set[str], scanned: set[str]) -> list[str]:
    """Each hit the baseline does not admit, and each entry, for a file scanned, no hit matches."""
    keys = {h.key for h in hits}
    out = [f"{h}: not in {BASELINE}" for h in hits if h.key not in baseline]
    for entry in sorted(baseline - keys):
        if entry.split()[0] in scanned:
            out.append(f"{BASELINE}: stale, no hit matches: {entry}")
    return out


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("files", nargs="*", help="files to scan (default: every testdata file)")
    parser.add_argument("--write-baseline", action="store_true", help="write the current hits as the baseline")
    args = parser.parse_args(argv)
    try:
        files = args.files or fixture_files(ROOT)
        hits = scan(ROOT, files)
        comments, baseline = read_baseline(ROOT / BASELINE)
    except (OSError, ValueError, subprocess.CalledProcessError) as err:
        print(f"fixtures-check: {err}", file=sys.stderr)
        return 2
    if args.write_baseline:
        if args.files:
            print("fixtures-check: --write-baseline scans every testdata file; name none", file=sys.stderr)
            return 2
        entries = sorted({h.key for h in hits})
        (ROOT / BASELINE).write_text("".join(c + "\n" for c in comments) + "".join(e + "\n" for e in entries),
                                     encoding="utf-8")
        print(f"fixtures-check: wrote {len(entries)} baseline line(s) for {len(hits)} hit(s)")
        return 0
    problems = findings(hits, baseline, set(files))
    for p in problems:
        print(f"fixtures-check: {p}")
    print(f"fixtures-check: {len(files)} file(s), {len(hits)} hit(s), {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
