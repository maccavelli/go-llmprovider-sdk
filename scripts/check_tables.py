#!/usr/bin/env python3
"""tables-check: the docs' import tables and live variables match the code.

Defined by docs/decisions/0046-MADR-v1-15-0-follow-ups.md D4 and A1; written
in 0046-PLAN Phase 1. It fails when:

  * a row of docs/architecture.md's package table names, in "Depends on",
    other packages than the package imports;
  * a package imports a package its row of R2, in
    docs/guides/api-standards.md, does not allow;
  * a package has no row in one of the two tables;
  * a live test (llmprovider/**/live_*_test.go) reads an LLMPROVIDER_LIVE_
    variable that AGENTS.md does not name, unless it is LLMPROVIDER_LIVE_<NNNN>
    with any suffix and a record numbered NNNN names it as a whole word;
  * AGENTS.md names a live variable that no live test reads.

The imports are `go list`'s, for the host and for GOOS=windows, merged; only
the module's own packages and golang.org/x/ are compared. A table cell's
names are its backticked tokens: a name is the package it equals, or the one
it ends in after a slash. "the above" (R2) is every package of the rows
above; "internal packages" every package with internal/ in its path; "the
provider packages" llmprovider/providers/<id>; "the standard library" adds
nothing. R2's rows for llmprovider/providers/<id> and
llmprovider/internal/wire/<format> cover the packages under them, except a
package with a row of its own.

Usage:
  scripts/check_tables.py

Exit codes: 0 clean, 1 a check failed, 2 tool error.
"""
from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
MODULE = "github.com/maccavelli/go-llmprovider-sdk"
ARCHITECTURE = "docs/architecture.md"
STANDARDS = "docs/guides/api-standards.md"
AGENTS = "AGENTS.md"
TICK = re.compile(r"`([^`]+)`")
LIVE_VAR = re.compile(r"LLMPROVIDER_LIVE_[A-Z0-9_]*[A-Z0-9]")
SWITCH = re.compile(r"^LLMPROVIDER_LIVE_(\d{4})")
PATTERNS = {
    "llmprovider/providers/<id>": re.compile(r"^llmprovider/providers/[^/]+$"),
    "llmprovider/internal/wire/<format>": re.compile(r"^llmprovider/internal/wire/[^/]+$"),
}


class ToolError(Exception):
    """A failure to read the code or a table, not a drift."""


def imports() -> dict[str, set[str]]:
    """Each package of the module, by path relative to it ("." for the root),
    with the module's packages and golang.org/x/ modules it imports."""
    out: dict[str, set[str]] = {}
    for goos in ("", "windows"):
        env = dict(os.environ, GOOS=goos) if goos else None
        proc = subprocess.run(["go", "list", "-f", "{{.ImportPath}}|{{join .Imports \",\"}}", "./..."],
                              cwd=ROOT, env=env, capture_output=True, text=True, check=False)
        if proc.returncode != 0:
            raise ToolError(f"GOOS={goos or 'host'} go list failed:\n{proc.stderr}")
        for line in proc.stdout.splitlines():
            path, _, deps = line.partition("|")
            got = out.setdefault(path.removeprefix(MODULE).lstrip("/") or ".", set())
            for dep in filter(None, deps.split(",")):
                if dep.startswith(MODULE + "/"):
                    got.add(dep.removeprefix(MODULE + "/"))
                elif dep.startswith("golang.org/x/"):
                    got.add(dep)
    return out


def table(path: str, header: str) -> list[tuple[str, str]]:
    """The first and last cells of each row of the table whose header row,
    without spaces, is header."""
    lines = (ROOT / path).read_text(encoding="utf-8").splitlines()
    starts = [i for i, line in enumerate(lines) if line.replace(" ", "").startswith(header)]
    if len(starts) != 1:
        raise ToolError(f"{path}: {len(starts)} tables headed {header!r}, want 1")
    rows = []
    for line in lines[starts[0] + 2:]:
        if not line.strip().startswith("|"):
            break
        cells = [c.strip() for c in line.strip().strip("|").split("|")]
        rows.append((cells[0], cells[-1]))
    return rows


def resolve(name: str, packages: set[str]) -> set[str] | None:
    """The package name names, or None when it names none or several."""
    if name in packages or name.startswith("golang.org/x/"):
        return {name}
    hits = {p for p in packages if p.endswith("/" + name)}
    return hits if len(hits) == 1 else None


def names(cell: str, packages: set[str], above: set[str]) -> tuple[set[str], list[str]]:
    """The packages a cell names, and the tokens that name none."""
    got: set[str] = set()
    unresolved = []
    for token in TICK.findall(cell):
        hit = resolve(token, packages)
        if hit is None:
            unresolved.append(token)
        else:
            got |= hit
    if "the above" in cell:
        got |= above
    if "internal packages" in cell:
        got |= {p for p in packages if p.startswith("internal/") or "/internal/" in p}
    if "the provider packages" in cell:
        got |= {p for p in packages if PATTERNS["llmprovider/providers/<id>"].match(p)}
    return got, unresolved


def check_architecture(pkgs: dict[str, set[str]]) -> list[str]:
    problems = []
    packages = set(pkgs)
    seen = set()
    for pkg_cell, deps_cell in table(ARCHITECTURE, "|Package|Holds|Dependson|"):
        tokens = TICK.findall(pkg_cell)
        pkg = tokens[0] if tokens else pkg_cell
        if pkg not in pkgs:
            problems.append(f"{ARCHITECTURE}: {pkg}: no such package")
            continue
        seen.add(pkg)
        listed, unresolved = names(deps_cell, packages, set())
        missing, stale = sorted(pkgs[pkg] - listed), sorted(listed - pkgs[pkg])
        if missing:
            problems.append(f"{ARCHITECTURE}: {pkg}: imports {', '.join(missing)}, which its row does not name")
        if stale:
            problems.append(f"{ARCHITECTURE}: {pkg}: names {', '.join(stale)}, which it does not import")
        if unresolved:
            problems.append(f"{ARCHITECTURE}: {pkg}: names no one package: {', '.join(unresolved)}")
    problems += [f"{ARCHITECTURE}: {p}: no row" for p in sorted(packages - seen)]
    return problems


def check_standards(pkgs: dict[str, set[str]]) -> list[str]:
    problems = []
    packages = set(pkgs)
    rows = []
    for pkg_cell, allow_cell in table(STANDARDS, "|Package|Holds|Mayimport|"):
        tokens = TICK.findall(pkg_cell)
        rows.append((tokens[0] if tokens else pkg_cell, allow_cell))
    own = {name for name, _ in rows if name in packages}
    above: set[str] = set()
    seen: set[str] = set()
    for name, allow_cell in rows:
        if name in PATTERNS:
            targets = {p for p in packages if PATTERNS[name].match(p) and p not in own}
        elif name in packages:
            targets = {name}
        else:
            problems.append(f"{STANDARDS}: {name}: no such package")
            continue
        allowed, unresolved = names(allow_cell, packages, above)
        if unresolved:
            problems.append(f"{STANDARDS}: {name}: names no one package: {', '.join(unresolved)}")
        for pkg in sorted(targets):
            extra = sorted(pkgs[pkg] - allowed - {pkg})
            if extra:
                problems.append(f"{STANDARDS}: {pkg}: imports {', '.join(extra)}, which its row does not allow")
        seen |= targets
        above |= targets
    problems += [f"{STANDARDS}: {p}: no row" for p in sorted(packages - seen)]
    return problems


def check_live_variables() -> list[str]:
    read: set[str] = set()
    for path in sorted((ROOT / "llmprovider").glob("**/live_*_test.go")):
        read |= set(LIVE_VAR.findall(path.read_text(encoding="utf-8")))
    named = set(LIVE_VAR.findall((ROOT / AGENTS).read_text(encoding="utf-8")))
    records = sorted((ROOT / "docs/decisions").glob("*.md")) + sorted((ROOT / "docs/reports").glob("*.md"))
    problems = []
    for var in sorted(read - named):
        m = SWITCH.match(var)
        if m:
            word = re.compile(r"(?<![A-Za-z0-9_])" + re.escape(var) + r"(?![A-Za-z0-9_])")
            if any(word.search(r.read_text(encoding="utf-8")) for r in records if r.name.startswith(m.group(1) + "-")):
                continue
        problems.append(f"{AGENTS}: {var}: a live test reads it, and {AGENTS} does not name it")
    problems += [f"{AGENTS}: {var}: named, and no live test reads it" for var in sorted(named - read)]
    return problems


def main() -> int:
    try:
        pkgs = imports()
        problems = check_architecture(pkgs) + check_standards(pkgs) + check_live_variables()
    except (ToolError, OSError) as err:
        print(f"check-tables: {err}", file=sys.stderr)
        return 2
    for p in problems:
        print(f"check-tables: {p}")
    print(f"check-tables: {len(pkgs)} packages, {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
