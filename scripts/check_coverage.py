#!/usr/bin/env python3
"""coverage-check: no package falls below its coverage floor.

Defined by docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md D13
and its amendment "the coverage of a test-support package", and the standards
guide's R47; written in 0015-PLAN S12. The floors are in
scripts/coverage-floors.txt: a package's P7 baseline, or 80 % for any package
not listed. A package is measured by its own tests, except a test-support
package, which the floors file says to measure by the tests that import it.
A package with no statements has no floor.

Usage:
  scripts/check_coverage.py

Exit codes: 0 clean, 1 a check failed, 2 tool or floors-file error.
"""
from __future__ import annotations

import re
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
FLOORS = ROOT / "scripts/coverage-floors.txt"
MODULE = "github.com/maccavelli/go-llmprovider-sdk"
DEFAULT_FLOOR = 80.0
COVER = re.compile(r"^(?:ok\s+)?\s*(\S+)\s.*coverage: (?:(\d+\.\d)% of statements|\[no statements\])")
TOTAL = re.compile(r"^total:\s+\(statements\)\s+(\d+\.\d)%$")


def run(*args: str) -> str:
    proc = subprocess.run(args, cwd=ROOT, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        print(f"coverage-check: {' '.join(args)} failed:\n{proc.stdout}{proc.stderr}", file=sys.stderr)
        sys.exit(2)
    return proc.stdout


def floors() -> dict[str, tuple[float, list[str]]]:
    out: dict[str, tuple[float, list[str]]] = {}
    for n, line in enumerate(FLOORS.read_text(encoding="utf-8").splitlines(), 1):
        fields = line.split("#", 1)[0].split()
        if not fields:
            continue
        try:
            out[fields[0]] = (float(fields[1]), fields[2:])
        except (IndexError, ValueError):
            print(f"coverage-check: {FLOORS.name}:{n}: want 'package floor [packages that measure it]'", file=sys.stderr)
            sys.exit(2)
    return out


def measured_by(pkg: str, importers: list[str]) -> float | None:
    """Coverage of pkg from the tests of importers (-coverpkg)."""
    with tempfile.TemporaryDirectory() as tmp:
        profile = Path(tmp) / "cover.out"
        run("go", "test", "-count=1", f"-coverpkg=./{pkg}", f"-coverprofile={profile}", *importers)
        for line in run("go", "tool", "cover", f"-func={profile}").splitlines():
            if m := TOTAL.match(line):
                return float(m.group(1))
    return None


def main() -> int:
    table = floors()
    own: dict[str, float | None] = {}
    for line in run("go", "test", "-count=1", "-cover", "./...").splitlines():
        if m := COVER.match(line):
            pkg = m.group(1).removeprefix(MODULE).lstrip("/") or "."
            own[pkg] = float(m.group(2)) if m.group(2) else None
    problems = [f"{FLOORS.name} names {p}, which is not a package" for p in sorted(table.keys() - own.keys())]
    for pkg in sorted(own):
        floor, importers = table.get(pkg, (DEFAULT_FLOOR, []))
        if importers and run("go", "list", "-f", "{{len .TestGoFiles}}{{len .XTestGoFiles}}", f"./{pkg}").strip() != "00":
            problems.append(f"{pkg} has test files, so it is measured by its own tests, not by its importers")
            importers = []
        got = measured_by(pkg, importers) if importers else own[pkg]
        if got is None:
            print(f"coverage-check: {pkg}: no statements")
            continue
        how = f" (by {' '.join(importers)})" if importers else ""
        verdict = "ok" if got >= floor else "BELOW"
        print(f"coverage-check: {pkg}: {got:.1f}% against {floor:.1f}%{how} {verdict}")
        if got < floor:
            problems.append(f"{pkg} is {got:.1f}%, below its {floor:.1f}% floor{how}")
    for p in problems:
        print(f"coverage-check: {p}")
    print(f"coverage-check: {len(own)} packages, {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
