#!/usr/bin/env python3
"""dep-check: only `wizard` leaves the standard library, and then only for
golang.org/x/term (and the golang.org/x/sys it needs).

Defined by docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md D2
and D13, and the standards guide's R2; written in 0015-PLAN S12. For every
package of the module it reads the transitive dependencies of its non-test
build and of its test build (`go list -deps`, then `go list -test -deps`),
and fails on one outside the standard library and this module that the
package is not allowed. It reads the build for the host and for
GOOS=windows, so an import in a _windows.go file is checked too (0010-PLAN
P6a). It also fails on a go.mod requirement outside REQUIRES (`go mod edit
-json`), so a module a test alone imports cannot slip in (0021-MADR Z4).

Usage:
  scripts/check_deps.py

Exit codes: 0 clean, 1 a check failed, 2 tool error.
"""
from __future__ import annotations

import json
import os
import re
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
# The modules go.mod may require (AGENTS.md "Dependencies").
REQUIRES = {"golang.org/x/term", "golang.org/x/sys"}
# A test build names its packages "path [path.test]", and its main package
# "path.test".
TEST_VARIANT = re.compile(r" \[[^\]]*\]$")


# The builds checked: the host's, and Windows's (empty means the host).
GOOSES = ("", "windows")


def go_list(goos: str, *args: str) -> list[str]:
    env = dict(os.environ, GOOS=goos) if goos else None
    proc = subprocess.run(["go", "list", *args], cwd=ROOT, env=env, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        print(f"dep-check: GOOS={goos or 'host'} go list {' '.join(args)} failed:\n{proc.stderr}", file=sys.stderr)
        sys.exit(2)
    return proc.stdout.splitlines()


def base(path: str) -> str:
    """A package path without its test-build decoration."""
    return TEST_VARIANT.sub("", path).removesuffix(".test")


def check(goos: str) -> tuple[int, list[str]]:
    """Returns the number of packages and the problems of one build, its
    tests included."""
    standard = {base(line.split()[0])
                for line in go_list(goos, "-deps", "-test", "-f", "{{.ImportPath}} {{.Standard}}", "./...")
                if line.endswith(" true")}
    problems = []
    count = len(go_list(goos, "-f", "{{.ImportPath}}", "./..."))
    for flags in ((), ("-test",)):
        for line in go_list(goos, *flags, "-f", "{{.ImportPath}}|{{join .Deps \",\"}}", "./..."):
            pkg, _, deps = line.partition("|")
            pkg = base(pkg)
            allowed = ALLOWED.get(pkg, ())
            for dep in map(base, filter(None, deps.split(","))):
                if dep in standard or dep == MODULE or dep.startswith(MODULE + "/"):
                    continue
                if not dep.startswith(allowed) or not allowed:
                    problem = f"GOOS={goos or 'host'}: {pkg.removeprefix(MODULE + '/')} depends on {dep}"
                    if problem not in problems:
                        problems.append(problem)
    return count, problems


def check_requires() -> list[str]:
    """Returns go.mod's requirements outside REQUIRES."""
    proc = subprocess.run(["go", "mod", "edit", "-json"], cwd=ROOT, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        print(f"dep-check: go mod edit -json failed:\n{proc.stderr}", file=sys.stderr)
        sys.exit(2)
    requires = json.loads(proc.stdout).get("Require") or []
    return [f"go.mod requires {r['Path']}" for r in requires if r["Path"] not in REQUIRES]


def main() -> int:
    problems = check_requires()
    print(f"dep-check: go.mod: {len(problems)} problem(s)")
    for goos in GOOSES:
        count, found = check(goos)
        problems += found
        print(f"dep-check: GOOS={goos or 'host'}: {count} packages, {len(found)} problem(s)")
    for p in problems:
        print(f"dep-check: {p}")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
