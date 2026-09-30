#!/usr/bin/env python3
"""G-parity: every exported identifier of mcplib v1.6.0's llmprovider and wizard
has a row in docs/guides/migrating-from-mcplib.md.

Defined by docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md D12
and 0015-PLAN S1. The baseline is docs/guides/migrating-from-mcplib.ids, one
identifier per line, generated from mcplib at v1.6.0 with the generate command.

Usage:
  scripts/check_parity_map.py [--require-equivalents]
      Check the guide against the baseline. Fails on an identifier with no
      row, on a row for an identifier not in the baseline, and, with
      --require-equivalents (on from 0015-PLAN S11), on an empty
      "SDK equivalent" cell.
  scripts/check_parity_map.py generate MCPLIB_DIR
      Print the baseline for an mcplib checkout (run `go doc -all` on its
      llmprovider and wizard packages).

Exit codes: 0 clean, 1 a check failed, 2 usage or tool error.
"""
from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
GUIDE = ROOT / "docs/guides/migrating-from-mcplib.md"
BASELINE = ROOT / "docs/guides/migrating-from-mcplib.ids"
PACKAGES = ("llmprovider", "wizard")
MAP_HEADING = "## Identifier map"

FUNC = re.compile(r"^func ([A-Z]\w*)[\[(]")
METHOD = re.compile(r"^func \([^)]*?\*?([A-Z]\w*)(?:\[[^\]]*\])?\) ([A-Z]\w*)[\[(]")
TYPE = re.compile(r"^type ([A-Z]\w*)\b")
STRUCT_OPEN = re.compile(r"^type ([A-Z]\w*)(?:\[[^\]]*\])? struct \{$")
IFACE_OPEN = re.compile(r"^type ([A-Z]\w*)(?:\[[^\]]*\])? interface \{$")
BLOCK_OPEN = re.compile(r"^(const|var) \($")
SINGLE = re.compile(r"^(?:const|var) ([A-Z]\w*(?:, [A-Z]\w*)*)\b")
MEMBER = re.compile(r"^\t([A-Z]\w*(?:, [A-Z]\w*)*)(?:\s|$|\()")
ROW = re.compile(r"^\|\s*`([^`]+)`\s*\|([^|]*)\|")


def identifiers(doc: str, package: str) -> set[str]:
    """Exported identifiers in `go doc -all` output: functions, types, methods,
    struct fields, interface methods, constants and variables."""
    found: set[str] = set()
    block: tuple[str, str] | None = None  # (kind, owner)
    for line in doc.splitlines():
        if block:
            if line in (")", "}"):
                block = None
                continue
            if line.lstrip().startswith("//"):
                continue
            m = MEMBER.match(line)
            if m:
                kind, owner = block
                names = [n.strip() for n in m.group(1).split(",")]
                if kind == "iface" and "(" not in line:
                    continue  # an embedded interface, not a method
                found.update(f"{owner}.{n}" if owner else n for n in names)
            continue
        if m := STRUCT_OPEN.match(line):
            found.add(m.group(1))
            block = ("struct", m.group(1))
        elif m := IFACE_OPEN.match(line):
            found.add(m.group(1))
            block = ("iface", m.group(1))
        elif m := BLOCK_OPEN.match(line):
            block = (m.group(1), "")
        elif m := METHOD.match(line):
            found.add(f"{m.group(1)}.{m.group(2)}")
        elif m := FUNC.match(line):
            found.add(m.group(1))
        elif m := TYPE.match(line):
            found.add(m.group(1))
        elif m := SINGLE.match(line):
            found.update(n.strip() for n in m.group(1).split(","))
    return {f"{package}.{name}" for name in found}


def generate(mcplib: Path) -> int:
    ids: set[str] = set()
    for package in PACKAGES:
        proc = subprocess.run(["go", "doc", "-all", f"./{package}"], cwd=mcplib,
                              capture_output=True, text=True, check=False)
        if proc.returncode != 0:
            print(f"go doc -all ./{package} failed:\n{proc.stderr}", file=sys.stderr)
            return 2
        ids |= identifiers(proc.stdout, package)
    print("\n".join(sorted(ids)))
    return 0


def rows(guide: str) -> dict[str, str]:
    """Identifier -> SDK-equivalent cell, from the tables under MAP_HEADING
    (up to the next level-2 heading). Other tables, such as import paths, are
    not identifier rows."""
    out: dict[str, str] = {}
    in_map = False
    for line in guide.splitlines():
        if line.startswith("## "):
            in_map = line.strip() == MAP_HEADING
            continue
        m = ROW.match(line) if in_map else None
        if m and "." in m.group(1):
            out[m.group(1).strip()] = m.group(2).strip()
    return out


def check(require_equivalents: bool) -> int:
    baseline = {l.strip() for l in BASELINE.read_text(encoding="utf-8").splitlines() if l.strip()}
    mapped = rows(GUIDE.read_text(encoding="utf-8"))
    problems = [f"no row: {i}" for i in sorted(baseline - mapped.keys())]
    problems += [f"row for an identifier not in the baseline: {i}" for i in sorted(mapped.keys() - baseline)]
    if require_equivalents:
        problems += [f"empty SDK equivalent: {i}" for i in sorted(baseline & mapped.keys()) if not mapped[i]]
    for p in problems:
        print(f"G-parity: {p}", file=sys.stderr)
    filled = sum(1 for i in baseline if mapped.get(i))
    print(f"G-parity: {len(baseline)} identifiers, {len(mapped)} rows, {filled} with an SDK equivalent, "
          f"{len(problems)} problem(s)")
    return 1 if problems else 0


def main(argv: list[str]) -> int:
    if argv[:1] == ["generate"] and len(argv) == 2:
        return generate(Path(argv[1]))
    if argv in ([], ["--require-equivalents"]):
        return check(bool(argv))
    print(__doc__, file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
