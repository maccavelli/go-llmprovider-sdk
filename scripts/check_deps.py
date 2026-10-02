#!/usr/bin/env python3
"""dep-check: only `wizard` leaves the standard library, and then only for
golang.org/x/term (and the golang.org/x/sys it needs).

Defined by docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md D2
and D13, and the standards guide's R2; written in 0015-PLAN S12. For every
package of the module it reads the transitive dependencies of its non-test
build (`go list -deps`), and fails on one outside the standard library and
this module that the package is not allowed.

Usage:
  scripts/check_deps.py

Exit codes: 0 clean, 1 a check failed, 2 tool error.
"""
from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
MODULE = "github.com/maccavelli/go-llmprovider-sdk"
# The packages allowed to leave the standard library, and the import-path
# prefixes each may reach.
ALLOWED = {
    f"{MODULE}/wizard": ("golang.org/x/term", "golang.org/x/sys/"),
}


def go_list(*args: str) -> list[str]:
    proc = subprocess.run(["go", "list", *args], cwd=ROOT, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        print(f"dep-check: go list {' '.join(args)} failed:\n{proc.stderr}", file=sys.stderr)
        sys.exit(2)
    return proc.stdout.splitlines()


def main() -> int:
    standard = {line.split()[0] for line in go_list("-deps", "-f", "{{.ImportPath}} {{.Standard}}", "./...")
                if line.endswith(" true")}
    problems = []
    packages = go_list("-f", "{{.ImportPath}}|{{join .Deps \" \"}}", "./...")
    for line in packages:
        pkg, _, deps = line.partition("|")
        allowed = ALLOWED.get(pkg, ())
        for dep in deps.split():
            if dep in standard or dep == MODULE or dep.startswith(MODULE + "/"):
                continue
            if not dep.startswith(allowed) or not allowed:
                problems.append(f"{pkg.removeprefix(MODULE + '/')} depends on {dep}")
    for p in problems:
        print(f"dep-check: {p}")
    print(f"dep-check: {len(packages)} packages, {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
