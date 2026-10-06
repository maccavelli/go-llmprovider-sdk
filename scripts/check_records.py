#!/usr/bin/env python3
"""records-check: the decision records and their index are consistent.

Written for 0025-PLAN (0025-MADR): the rules AGENTS.md ("Records") and the
madr-and-plan-writing skill set for docs/decisions/ and docs/reports/, checked
rather than remembered. It checks:

- names: NNNN-{MADR,PLAN}-slug.md in docs/decisions/, NNNN-{REPORT,GATES}-
  slug.md in docs/reports/, the slug lowercase kebab-case, and no other .md
  file in either directory;
- numbers: at most one MADR per number;
- statuses: a MADR's front-matter status is proposed, accepted, rejected,
  deprecated, or begins "superseded"; a PLAN's is proposed, in-progress,
  complete or superseded; a REPORT or GATES has none, or observation;
- pairs: every PLAN has a MADR of its number, a number's only PLAN has its
  MADR's slug, and a PLAN's associated-madr or parent-madr names that MADR;
- the index: docs/README.md's "Records" section has one row per record, whose
  number, kind and link match the file, whose status is the file's (or
  "observation" for a report with none), and whose "N records." line is both
  the number of records and the number of rows.

With --next it prints only the next record number, the highest across all
four kinds plus one, and checks nothing, so a broken index does not block
numbering.

Usage:
  scripts/check_records.py
  scripts/check_records.py --next

Exit codes: 0 clean, 1 a check failed.
"""
from __future__ import annotations

import re
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DIRS = {"docs/decisions": ("MADR", "PLAN"), "docs/reports": ("REPORT", "GATES")}
NAME = re.compile(r"^(\d{4})-(MADR|PLAN|REPORT|GATES)-([a-z0-9]+(?:-[a-z0-9]+)*)\.md$")
MADR_STATUSES = ("proposed", "accepted", "rejected", "deprecated")
PLAN_STATUSES = ("proposed", "in-progress", "complete", "superseded")
REPORT_STATUS = "observation"
INDEX = "docs/README.md"
ROW = re.compile(r"^\| (\d{4}) \| ([A-Z]+) \| \[.*\]\(((?:decisions|reports)/[^)\s]+)\) \| ([^|]*?) \|\s*$")
COUNT = re.compile(r"^(\d+) records\.", re.M)


@dataclass
class Record:
    path: str  # relative to ROOT
    number: str
    kind: str
    slug: str
    meta: dict[str, str]


def front_matter(text: str) -> dict[str, str]:
    """The plain key: value lines of a leading --- block, unquoted."""
    m = re.match(r"^---\n(.*?)\n---\n", text, re.S)
    meta: dict[str, str] = {}
    if m:
        for line in m.group(1).splitlines():
            key, sep, value = line.partition(":")
            if sep and not line.startswith((" ", "#")):
                meta[key.strip()] = value.strip().strip("\"'")
    return meta


def scan(problems: list[str]) -> list[Record]:
    records = []
    for d, kinds in DIRS.items():
        for p in sorted((ROOT / d).glob("*.md")):
            rel = f"{d}/{p.name}"
            m = NAME.match(p.name)
            if not m:
                problems.append(f"{rel}: name is not NNNN-KIND-slug.md")
                continue
            if m.group(2) not in kinds:
                problems.append(f"{rel}: a {m.group(2)} belongs in {next(k for k, v in DIRS.items() if m.group(2) in v)}/")
            records.append(Record(rel, m.group(1), m.group(2), m.group(3), front_matter(p.read_text(encoding="utf-8"))))
    return records


def check_records(records: list[Record], problems: list[str]) -> None:
    madrs: dict[str, list[Record]] = {}
    plans: dict[str, list[Record]] = {}
    for r in records:
        (madrs if r.kind == "MADR" else plans if r.kind == "PLAN" else {}).setdefault(r.number, []).append(r)
    for number, group in madrs.items():
        if len(group) > 1:
            for r in group:
                problems.append(f"{r.path}: one of {len(group)} MADRs numbered {number}")
    for r in records:
        status = r.meta.get("status")
        if r.kind == "MADR" and not (status in MADR_STATUSES or (status or "").startswith("superseded")):
            problems.append(f"{r.path}: MADR status {status!r}, want one of {', '.join(MADR_STATUSES)} or superseded")
        elif r.kind == "PLAN" and status not in PLAN_STATUSES:
            problems.append(f"{r.path}: PLAN status {status!r}, want one of {', '.join(PLAN_STATUSES)}")
        elif r.kind in ("REPORT", "GATES") and status not in (None, REPORT_STATUS):
            problems.append(f"{r.path}: {r.kind} status {status!r}, want none or {REPORT_STATUS}")
    for number, group in plans.items():
        madr = madrs.get(number, [None])[0]
        for r in group:
            if madr is None:
                problems.append(f"{r.path}: no MADR numbered {number}")
                continue
            if len(group) == 1 and r.slug != madr.slug:
                problems.append(f"{r.path}: the only PLAN numbered {number} has slug {r.slug!r}; its MADR's is {madr.slug!r}")
            named = r.meta.get("associated-madr") or r.meta.get("parent-madr")
            if named is None:
                problems.append(f"{r.path}: front matter names no associated-madr or parent-madr")
            elif Path(named).name != Path(madr.path).name:
                problems.append(f"{r.path}: names MADR {named!r}; its number's MADR is {Path(madr.path).name!r}")


def check_index(records: list[Record], problems: list[str]) -> None:
    text = (ROOT / INDEX).read_text(encoding="utf-8")
    section = re.search(r"^## Records\n(.*?)(?=^## |\Z)", text, re.M | re.S)
    if section is None:
        problems.append(f"{INDEX}: no \"## Records\" section")
        return
    body = section.group(1)
    count = COUNT.search(body)
    if count is None:
        problems.append(f"{INDEX}: no \"N records.\" line in the Records section")
    elif int(count.group(1)) != len(records):
        problems.append(f"{INDEX}: says {count.group(1)} records; there are {len(records)}")
    by_link = {r.path.removeprefix("docs/"): r for r in records}
    seen: set[str] = set()
    for line in body.splitlines():
        row = ROW.match(line)
        if not row:
            continue
        number, kind, link, status = row.groups()
        r = by_link.get(link)
        if r is None:
            problems.append(f"{INDEX}: row {number} {kind} links {link}, which is not a record")
            continue
        if link in seen:
            problems.append(f"{INDEX}: {link} is indexed twice")
        seen.add(link)
        if (number, kind) != (r.number, r.kind):
            problems.append(f"{INDEX}: row {number} {kind} links {link}, a {r.number} {r.kind}")
        want = r.meta.get("status") or (REPORT_STATUS if r.kind in ("REPORT", "GATES") else None)
        if status.strip() != want:
            problems.append(f"{INDEX}: {link} indexed as {status.strip()!r}; the file says {want!r}")
    for link, r in by_link.items():
        if link not in seen:
            problems.append(f"{r.path}: not indexed in {INDEX}")
    rows = sum(1 for line in body.splitlines() if ROW.match(line))
    if count is not None and int(count.group(1)) != rows:
        problems.append(f"{INDEX}: says {count.group(1)} records; its Records table has {rows} rows")


def next_number() -> str:
    numbers = [int(m.group(1)) for d in DIRS for p in (ROOT / d).glob("*.md") if (m := NAME.match(p.name))]
    return f"{max(numbers, default=0) + 1:04d}"


def main() -> int:
    if sys.argv[1:] == ["--next"]:
        print(next_number())
        return 0
    if sys.argv[1:]:
        print(__doc__.split("Usage:")[1].split("Exit")[0].strip(), file=sys.stderr)
        return 1
    problems: list[str] = []
    records = scan(problems)
    check_records(records, problems)
    check_index(records, problems)
    for p in sorted(problems):
        print(p)
    print(f"records-check: {len(records)} records, {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
