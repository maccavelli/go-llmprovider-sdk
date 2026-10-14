#!/usr/bin/env python3
"""claims: each row of a record's evidence index holds its text, on its line, at its revision.

Written for docs/decisions/0040-PLAN-native-tool-deferral.md Phase 0 and
0041-PLAN-decide-per-model-behaviour-from-metadata.md Phase 0, under
docs/decisions/0035-MADR-maintainer-tools-in-python.md. It is run by a person
or an agent before a record is staged, never by the gate: a record's claims
are pinned to the revisions it names, which the gate's copy does not hold.

A row is a Markdown table row whose first cell is an id such as E12 or P3,
whose third cell is the location in backticks, and whose last cell is the
matched text in backticks:

  | E12 | what is claimed | `path/file.go:403` | `func claudeAdaptiveOnly(` |
  | E36 | another repository | `other-repo:dir/file.go:58` | `text` |

The record names each revision once, in a comment alone on its line:

  <!-- claims: self=3e81e3d other-repo=9b75e5f3 -->

`self` is this repository. Another repository is found as a directory of that
name beside this one, or beside its parent, unless --repo names it. Its file
is read with `git show REV:PATH`, never from the working tree.

Usage:
  tools/claims.py RECORD... [--repo NAME=DIR]... [--rev NAME=REV]...
                  [--search DIR]... [--json] [--fixes SPEC.json]
  tools/claims.py --changed [same options]

--changed checks the records under docs/ that `git status` lists as added,
changed or untracked. --json prints one JSON object instead of text. --fixes
writes a tools/edit.py spec that moves each failing row to the one line where
its text was found, for a person or an agent to review and apply.

Exit codes: 0 every row passes, 1 a row fails, a repository or a revision is
missing, or a record has no rows, 2 tool error.
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from dataclasses import asdict, dataclass, field
from pathlib import Path

from _common import ROOT

ROW = re.compile(r"^\|\s*([A-Z]\d+)\s*\|.*\|\s*`([^`]+)`\s*\|\s*(.+?)\s*\|\s*$")
# A marker stands alone on its line, so prose that quotes the syntax, such as
# `<!-- claims: NAME=REV -->` in backticks, is not read as one.
MARKER = re.compile(r"^[ \t]*<!--\s*claims:\s*(.*?)\s*-->[ \t]*$", re.M)
RECORD_DIRS = ("docs/decisions/", "docs/reports/")


@dataclass
class Result:
    """One row's outcome."""

    id: str
    where: str
    status: str  # pass, fail, missing
    detail: str = ""
    found: list[int] = field(default_factory=list)


def matched_text(cell: str) -> str:
    """The text of a code cell: `x`, or `` x `` for text holding a backtick, with \\| unescaped."""
    cell = cell.strip()
    m = re.fullmatch(r"``\s(.+)\s``", cell) or re.fullmatch(r"`(.+)`", cell)
    if not m:
        raise ValueError(f"the matched cell is not code: {cell!r}")
    return m.group(1).replace("\\|", "|")


def revisions(text: str) -> dict[str, str]:
    """The NAME=REV pairs of a record's claims marker."""
    m = MARKER.search(text)
    if not m:
        return {}
    pairs = {}
    for item in m.group(1).split():
        name, sep, rev = item.partition("=")
        if not sep or not name or not rev:
            raise ValueError(f"claims marker: {item!r} is not NAME=REV")
        pairs[name] = rev
    return pairs


def find_repo(name: str, repos: dict[str, Path], search: list[Path]) -> Path | None:
    """The checkout of name: --repo's, else the first search directory holding name/.git."""
    if name in repos:
        return repos[name]
    if name == "self":
        return ROOT
    for base in search:
        candidate = base / name
        if (candidate / ".git").exists():
            return candidate
    return None


class Files:
    """Files read at a revision, each read once."""

    def __init__(self) -> None:
        self._cache: dict[tuple[Path, str, str], list[str] | str] = {}

    def lines(self, repo: Path, rev: str, path: str) -> list[str] | str:
        """The file's lines, or the git error as a string."""
        key = (repo, rev, path)
        if key not in self._cache:
            p = subprocess.run(["git", "-C", str(repo), "show", f"{rev}:{path}"], capture_output=True, text=True)
            self._cache[key] = p.stdout.splitlines() if p.returncode == 0 else p.stderr.strip()
        return self._cache[key]


def check_record(path: Path, repos: dict[str, Path], revs: dict[str, str], search: list[Path],
                 files: Files) -> tuple[list[Result], list[str], str]:
    """Check one record: its results, its raw rows (for --fixes), and an error for the record."""
    text = path.read_text(encoding="utf-8")
    try:
        marker = {**revisions(text), **revs}
    except ValueError as err:
        return [], [], str(err)
    results, raw = [], []
    for line in text.splitlines():
        m = ROW.match(line)
        if not m:
            continue
        rid, where = m.group(1), m.group(2)
        raw.append(line)
        try:
            want = matched_text(m.group(3))
        except ValueError as err:
            results.append(Result(rid, where, "fail", str(err)))
            continue
        parts = where.split(":")
        if len(parts) not in (2, 3) or not parts[-1].isdigit():
            results.append(Result(rid, where, "fail", "the location is not path:line or name:path:line"))
            continue
        name = parts[0] if len(parts) == 3 else "self"
        rel, num = parts[-2], int(parts[-1])
        repo = find_repo(name, repos, search)
        rev = marker.get(name)
        if repo is None:
            results.append(Result(rid, where, "missing", f"no checkout of {name!r}: pass --repo {name}=DIR"))
            continue
        if rev is None:
            results.append(Result(rid, where, "missing", f"no revision for {name!r}: add it to the claims marker or pass --rev {name}=REV"))
            continue
        lines = files.lines(repo, rev, rel)
        if isinstance(lines, str):
            results.append(Result(rid, where, "fail", f"{name}@{rev}:{rel}: {lines}"))
            continue
        got = lines[num - 1] if 0 < num <= len(lines) else ""
        if want in got:
            results.append(Result(rid, where, "pass"))
            continue
        found = [i + 1 for i, text_line in enumerate(lines) if want in text_line]
        results.append(Result(rid, where, "fail", f"line {num} is {got.strip()[:90]!r}", found))
    return results, raw, ""


def shown(path: Path) -> str:
    """path relative to the repository when it is inside it, so no machine path is printed."""
    return str(path.relative_to(ROOT)) if path.is_relative_to(ROOT) else str(path)


def fixes_for(record: Path, results: list[Result], raw: list[str]) -> list[dict[str, str]]:
    """An edit.py entry per failing row whose text is on exactly one other line."""
    entries = []
    by_id = {ROW.match(line).group(1): line for line in raw}
    for r in results:
        if r.status != "fail" or len(r.found) != 1:
            continue
        old = by_id[r.id]
        new_where = r.where.rsplit(":", 1)[0] + f":{r.found[0]}"
        entries.append({"file": shown(record), "old": old, "new": old.replace(f"`{r.where}`", f"`{new_where}`", 1)})
    return entries


def changed_records() -> list[Path]:
    """Records under docs/ that git status lists as added, changed or untracked."""
    p = subprocess.run(["git", "-C", str(ROOT), "status", "--porcelain", "--untracked-files=all"],
                       capture_output=True, text=True, check=True)
    out = []
    for line in p.stdout.splitlines():
        name = line[3:].split(" -> ")[-1]
        if name.startswith(RECORD_DIRS) and name.endswith(".md") and (ROOT / name).exists():
            out.append(ROOT / name)
    return sorted(out)


def pairs(values: list[str], what: str) -> dict[str, str]:
    out = {}
    for v in values:
        name, sep, rest = v.partition("=")
        if not sep or not name or not rest:
            raise ValueError(f"{what} {v!r} is not NAME=VALUE")
        out[name] = rest
    return out


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("records", nargs="*", type=Path)
    parser.add_argument("--changed", action="store_true", help="check the records git status lists")
    parser.add_argument("--repo", action="append", default=[], help="NAME=DIR, a checkout of another repository")
    parser.add_argument("--rev", action="append", default=[], help="NAME=REV, over the record's marker")
    parser.add_argument("--search", action="append", default=[], type=Path,
                        help="a directory holding other checkouts (default: this repository's parent and its parent)")
    parser.add_argument("--json", action="store_true", help="print one JSON object")
    parser.add_argument("--fixes", type=Path, help="write an edit.py spec for rows found on another line")
    args = parser.parse_args(argv)
    try:
        repos = {k: Path(v) for k, v in pairs(args.repo, "--repo").items()}
        revs = pairs(args.rev, "--rev")
    except ValueError as err:
        print(f"claims: {err}", file=sys.stderr)
        return 2
    search = args.search or [ROOT.parent, ROOT.parent.parent]
    records = [r if r.is_absolute() else Path.cwd() / r for r in args.records]
    if args.changed:
        records += changed_records()
    if not records:
        print("claims: no record given", file=sys.stderr)
        return 2

    files, report, fixes, finding = Files(), [], [], False
    for record in records:
        if not record.is_file():
            print(f"claims: {record}: not a file", file=sys.stderr)
            return 2
        results, raw, err = check_record(record, repos, revs, search, files)
        passed = sum(r.status == "pass" for r in results)
        if err or not results or passed != len(results):
            finding = True
        fixes += fixes_for(record, results, raw)
        report.append({"record": shown(record), "error": err or ("no evidence rows" if not results else ""),
                       "passed": passed, "total": len(results), "rows": [asdict(r) for r in results]})

    if args.fixes:
        args.fixes.write_text(json.dumps(fixes, indent=2) + "\n", encoding="utf-8")
    if args.json:
        print(json.dumps({"ok": not finding, "fixes": len(fixes), "records": report}, indent=2))
    else:
        for rec in report:
            for row in rec["rows"]:
                if row["status"] != "pass":
                    hint = f"; text found at {row['found']}" if row["found"] else ""
                    print(f"{row['status'].upper()} {row['id']} {row['where']}: {row['detail']}{hint}")
            note = f" ({rec['error']})" if rec["error"] else ""
            print(f"claims: {rec['record']}: {rec['passed']} of {rec['total']} pass{note}")
        if args.fixes:
            print(f"claims: {len(fixes)} fix(es) written to {args.fixes}")
    return 1 if finding else 0


if __name__ == "__main__":
    sys.exit(main())
