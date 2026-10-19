#!/usr/bin/env python3
"""records: a record's status, and its row in docs/README.md, changed together.

Defined by docs/decisions/0047-MADR-keep-five-more-scratch-helpers-as-tools.md
D3. Two commands:

  add RECORD...                       each record's row is added to the index's
                                      Records table, after the last row of the
                                      same or a lower number: its number, kind,
                                      a link titled with its first heading, and
                                      its status; "N records." is updated.
  status RECORD STATUS [--date DATE]  the record's front-matter status and date
                                      (today by default), and its row's status.

The statuses allowed, the index's path, its row and its count line are
scripts/check_records.py's, imported, so the two never disagree. After
writing, it runs check_records.py's checks. It never touches the "I want to…"
table, which a person writes.

Usage:
  tools/records.py [--root DIR] add RECORD...
  tools/records.py [--root DIR] status RECORD STATUS [--date YYYY-MM-DD]

Exit codes: 0 written and the checks pass, 1 a check failed or the index
already holds (or lacks) the row, 2 tool error, a bad status or a bad record.
"""
from __future__ import annotations

import argparse
import contextlib
import datetime
import re
import sys
from collections.abc import Iterator
from pathlib import Path

from _common import ROOT

sys.path.insert(0, str(ROOT / "scripts"))
import check_records  # noqa: E402

HEADING = re.compile(r"^# (.+?)\s*$", re.M)


class Refused(Exception):
    """A change the index cannot take: exit 1."""


@contextlib.contextmanager
def checking(root: Path) -> Iterator[None]:
    """check_records reads its module's ROOT: point it at root for the block."""
    saved, check_records.ROOT = check_records.ROOT, root
    try:
        yield
    finally:
        check_records.ROOT = saved


def describe(root: Path, record: Path) -> tuple[str, str, str, str, str]:
    """The record's number, kind, index link, title and indexed status."""
    path = record if record.is_absolute() else root / record
    m = check_records.NAME.match(path.name)
    if m is None:
        raise ValueError(f"{record}: not a NNNN-KIND-slug.md record")
    number, kind = m.group(1), m.group(2)
    text = path.read_text(encoding="utf-8")
    body = re.sub(r"^---\n.*?\n---\n", "", text, count=1, flags=re.S)
    heading = HEADING.search(body)
    if heading is None:
        raise ValueError(f"{record}: no first heading for its title")
    status = check_records.front_matter(text).get("status")
    if status is None and kind in ("REPORT", "GATES"):
        status = check_records.REPORT_STATUS
    if status is None:
        raise ValueError(f"{record}: no front-matter status")
    folder = "decisions" if kind in ("MADR", "PLAN") else "reports"
    return number, kind, f"{folder}/{path.name}", heading.group(1), status


def records_section(text: str) -> tuple[int, int]:
    """The start and end offsets of the index's Records section body."""
    m = re.search(r"^## Records\n(.*?)(?=^## |\Z)", text, re.M | re.S)
    if m is None:
        raise ValueError(f"{check_records.INDEX}: no \"## Records\" section")
    return m.start(1), m.end(1)


def recount(body: str) -> str:
    """body with its "N records." line set to its number of rows."""
    rows = sum(1 for line in body.splitlines() if check_records.ROW.match(line))
    if check_records.COUNT.search(body) is None:
        raise ValueError(f"{check_records.INDEX}: no \"N records.\" line")
    return check_records.COUNT.sub(f"{rows} records.", body, count=1)


def add(root: Path, records: list[Path]) -> None:
    index = root / check_records.INDEX
    text = index.read_text(encoding="utf-8")
    start, end = records_section(text)
    lines = text[start:end].split("\n")
    for record in records:
        number, kind, link, title, status = describe(root, record)
        rows = [(i, check_records.ROW.match(line)) for i, line in enumerate(lines)]
        rows = [(i, m) for i, m in rows if m]
        if any(m.group(3) == link for _, m in rows):
            raise Refused(f"{link} is already indexed")
        if not rows:
            raise ValueError(f"{check_records.INDEX}: the Records table has no rows to place a row among")
        after = max((i for i, m in rows if m.group(1) <= number), default=rows[0][0] - 1)
        lines.insert(after + 1, f"| {number} | {kind} | [{title}]({link}) | {status} |")
    index.write_text(text[:start] + recount("\n".join(lines)) + text[end:], encoding="utf-8")


def set_status(root: Path, record: Path, status: str, date: str) -> None:
    path = record if record.is_absolute() else root / record
    m = check_records.NAME.match(path.name)
    if m is None:
        raise ValueError(f"{record}: not a NNNN-KIND-slug.md record")
    kind = m.group(2)
    allowed = {"MADR": check_records.MADR_STATUSES, "PLAN": check_records.PLAN_STATUSES}.get(
        kind, (check_records.REPORT_STATUS,))
    if status not in allowed and not (kind == "MADR" and status.startswith("superseded")):
        raise ValueError(f"{record}: a {kind}'s status is one of {', '.join(allowed)}, not {status!r}")
    datetime.date.fromisoformat(date)
    text = path.read_text(encoding="utf-8")
    fm = re.match(r"^---\n(.*?)\n---\n", text, re.S)
    if fm is None or not re.search(r"^status:", fm.group(1), re.M):
        raise ValueError(f"{record}: its front matter has no status line")
    block = re.sub(r"^status:.*$", f"status: {status}", fm.group(1), count=1, flags=re.M)
    if re.search(r"^date:", block, re.M):
        block = re.sub(r"^date:.*$", f"date: {date}", block, count=1, flags=re.M)
    else:
        block = re.sub(r"^(status:.*)$", rf"\1\ndate: {date}", block, count=1, flags=re.M)
    new_text = f"---\n{block}\n---\n" + text[fm.end():]

    index = root / check_records.INDEX
    index_text = index.read_text(encoding="utf-8")
    start, end = records_section(index_text)
    link = f"{'decisions' if kind in ('MADR', 'PLAN') else 'reports'}/{path.name}"
    lines = index_text[start:end].split("\n")
    hits = [i for i, line in enumerate(lines) if (row := check_records.ROW.match(line)) and row.group(3) == link]
    if len(hits) != 1:
        raise Refused(f"{link} is indexed {len(hits)} time(s); add it first")
    i = hits[0]
    lines[i] = re.sub(r"\| [^|]*? \|\s*$", f"| {status} |", lines[i])
    path.write_text(new_text, encoding="utf-8")
    index.write_text(index_text[:start] + "\n".join(lines) + index_text[end:], encoding="utf-8")


def check(root: Path) -> list[str]:
    """check_records.py's problems, for root."""
    problems: list[str] = []
    with checking(root):
        found = check_records.scan(problems)
        check_records.check_records(found, problems)
        check_records.check_index(found, problems)
    return sorted(problems)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository (default: this one)")
    sub = parser.add_subparsers(dest="command", required=True)
    p_add = sub.add_parser("add", help="index records")
    p_add.add_argument("records", nargs="+", type=Path)
    p_status = sub.add_parser("status", help="set a record's status")
    p_status.add_argument("record", type=Path)
    p_status.add_argument("status")
    p_status.add_argument("--date", default=datetime.date.today().isoformat())
    try:
        args = parser.parse_args(argv)
    except SystemExit as exit_:
        return 0 if exit_.code == 0 else 2
    root = args.root.resolve()
    try:
        if args.command == "add":
            add(root, args.records)
        else:
            set_status(root, args.record, args.status, args.date)
    except Refused as err:
        print(f"records: {err}; nothing written")
        return 1
    except (OSError, ValueError) as err:
        print(f"records: {err}; nothing written", file=sys.stderr)
        return 2
    problems = check(root)
    for p in problems:
        print(p)
    print(f"records: written; records-check: {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
