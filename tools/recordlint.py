#!/usr/bin/env python3
"""recordlint: the markdownlint findings an edit adds to a record.

Defined by docs/decisions/0047-MADR-keep-five-more-scratch-helpers-as-tools.md
D4 and A2. The gate's markdownlint skips the records, and many records carry
findings older than any edit, so a plain run is noise. For each record, this
lints the file at HEAD and in the tree, each from a copy in a temporary
directory, and compares the count of each rule.

The config is the repository's (.markdownlint-cli2.jsonc) with two rules set
for the records' own conventions (A2): MD004 "consistent", since records use
"*" list markers, and MD010 with code_blocks false, since records quote Go,
whose indent is a tab. Copies are linted, not the files in place, because
markdownlint-cli2 would otherwise read the repository's config from the
record's directory over this one. A record's own markdownlint-disable
comments are kept, as markdownlint keeps them.

Usage:
  tools/recordlint.py RECORD...
  tools/recordlint.py --changed

--changed lints the records under docs/decisions/ and docs/reports/ that git
status lists as added, changed or untracked. A record new since HEAD is
compared with no findings.

Exit codes: 0 no rule's count rose, 1 a rule's count rose, 2 tool error
(markdownlint-cli2 not on PATH, a record not found).
"""
from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import sys
import tempfile
from collections import Counter
from pathlib import Path

from _common import ROOT

RECORD_DIRS = ("docs/decisions/", "docs/reports/")
OVERRIDES = {"MD004": {"style": "consistent"}, "MD010": {"code_blocks": False}}
FINDING = re.compile(r"^(?P<file>[^:\s]+\.md):(?P<line>\d+)(?::\d+)? error (?P<rule>MD\d+)\S*\s*(?P<text>.*)$")


def records_config(root: Path) -> dict:
    """The repository's markdownlint rules, with the records' two (A2)."""
    path = root / ".markdownlint-cli2.jsonc"
    rules: dict = {}
    if path.exists():
        text = re.sub(r"/\*.*?\*/", "", path.read_text(encoding="utf-8"), flags=re.S)
        text = re.sub(r"^\s*//.*$", "", text, flags=re.M)
        text = re.sub(r",(\s*[}\]])", r"\1", text)
        rules = json.loads(text).get("config", {})
    return {"config": {**rules, **OVERRIDES}}


def lint(directory: Path, names: list[str]) -> list[tuple[str, int, str, str]]:
    """markdownlint-cli2's findings for names in directory: (file, line, rule, text)."""
    if not names:
        return []
    proc = subprocess.run(["markdownlint-cli2", "--no-globs", *names], cwd=directory,
                          capture_output=True, text=True)
    out = []
    for line in (proc.stdout + proc.stderr).splitlines():
        if m := FINDING.match(line.strip()):
            out.append((m["file"], int(m["line"]), m["rule"], m["text"]))
    if proc.returncode not in (0, 1):
        raise RuntimeError(f"markdownlint-cli2 exit {proc.returncode}: {proc.stderr.strip()[:200]}")
    return out


def changed_records(root: Path) -> list[str]:
    out = subprocess.run(["git", "status", "--porcelain", "--untracked-files=all", "--", *RECORD_DIRS],
                         cwd=root, capture_output=True, text=True, check=True).stdout
    paths = [line[3:].split(" -> ")[-1] for line in out.splitlines() if line[:2].strip() != "D"]
    return sorted(p for p in paths if p.endswith(".md"))


def at_head(root: Path, rel: str) -> str | None:
    """The record at HEAD, or None if it is new."""
    p = subprocess.run(["git", "show", f"HEAD:{rel}"], cwd=root, capture_output=True, text=True)
    return p.stdout if p.returncode == 0 else None


def compare(root: Path, records: list[str]) -> tuple[list[str], bool]:
    """The report's lines, and whether any rule's count rose."""
    config = json.dumps(records_config(root))
    work = Path(tempfile.mkdtemp(prefix="recordlint-"))
    try:
        head, tree = work / "head", work / "tree"
        for d in (head, tree):
            d.mkdir()
            (d / ".markdownlint-cli2.jsonc").write_text(config, encoding="utf-8")
        names: dict[str, str] = {}
        head_names = []
        for rel in records:
            name = f"{len(names):03d}-{Path(rel).name}"
            names[name] = rel
            shutil.copyfile(root / rel, tree / name)
            old = at_head(root, rel)
            if old is not None:
                (head / name).write_text(old, encoding="utf-8")
                head_names.append(name)
        before, after = lint(head, head_names), lint(tree, list(names))
    finally:
        shutil.rmtree(work, ignore_errors=True)
    lines, rose = [], False
    for name, rel in names.items():
        b = Counter(rule for f, _, rule, _ in before if f == name)
        a = Counter(rule for f, _, rule, _ in after if f == name)
        risen = sorted(r for r in a if a[r] > b[r])
        rose = rose or bool(risen)
        verdict = "new findings" if risen else "no new findings"
        lines.append(f"{rel}: {verdict}; HEAD {dict(sorted(b.items())) or '{}'}, tree {dict(sorted(a.items())) or '{}'}")
        for f, line, rule, text in after:
            if f == name and rule in risen:
                lines.append(f"  {rel}:{line} {rule} {text}")
    return lines, rose


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("records", nargs="*", help="records, relative to the repository")
    parser.add_argument("--changed", action="store_true", help="lint the records git status lists")
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository (default: this one)")
    args = parser.parse_args(argv)
    root = args.root.resolve()
    if shutil.which("markdownlint-cli2") is None:
        print("recordlint: markdownlint-cli2 is not on PATH", file=sys.stderr)
        return 2
    try:
        records = changed_records(root) if args.changed else list(args.records)
        missing = [r for r in records if not (root / r).is_file()]
        if missing:
            raise FileNotFoundError(f"not found: {', '.join(missing)}")
        lines, rose = compare(root, records)
    except (OSError, RuntimeError, ValueError, subprocess.CalledProcessError) as err:
        print(f"recordlint: {err}", file=sys.stderr)
        return 2
    for line in lines:
        print(line)
    print(f"recordlint: {len(records)} record(s); {'a rule rose' if rose else 'no rule rose'}")
    return 1 if rose else 0


if __name__ == "__main__":
    sys.exit(main())
