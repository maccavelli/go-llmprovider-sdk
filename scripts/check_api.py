#!/usr/bin/env python3
"""api-check: no incompatible API change against the latest v1 release.

Defined by docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md D13
and the standards guide's R48; written in 0015-PLAN S12. It compares the
module's exported API with the latest `v1.X.Y` tag (release candidates do not
count) using apidiff at a pinned version, run with `go run`: a tool, not a
module requirement. Internal packages are not compared, and changes under
llmprovider/x/, the experimental tree, are reported without failing.

apidiff runs once for each subtree in SUBTREES, not once for the module: with
the module's own nested module authcontract in its build list (0074-PLAN
Phase 4), the pattern `<module>/...` also matches that module's packages,
which sort first, and apidiff files the whole export under the nested
module's path. A root-module package outside the subtrees, other than an
internal one, would go uncompared, so it fails the gate (exit 1).

Before the first v1 release tag there is nothing to compare: it says so and
passes.

Usage:
  scripts/check_api.py

Exit codes: 0 clean, 1 an incompatible change, 2 tool error.
"""
from __future__ import annotations

import io
import re
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
MODULE = "github.com/maccavelli/go-llmprovider-sdk"
APIDIFF = "golang.org/x/exp/cmd/apidiff@v0.0.0-20260908205506-85c1c2202aba"
# The subtrees of the root module that apidiff compares, one run each. Every
# public package of the module is under one of them (see uncovered).
SUBTREES = ("llmprovider", "wizard")
RELEASE = re.compile(r"^v1\.(\d+)\.(\d+)$")
# A change under llmprovider/x/, as apidiff -m names it: an identifier
# ("./llmprovider/x/demo.Gone: removed") or a whole package ("package
# <module>/llmprovider/x/demo: removed") (0020-MADR F54).
EXPERIMENTAL = re.compile(r"^(?:\./|package " + re.escape(MODULE) + r"/)llmprovider/x[/.:]")


def run(args: list[str], cwd: Path, binary: bool = False) -> str | bytes:
    proc = subprocess.run(args, cwd=cwd, capture_output=True, text=not binary, check=False)
    if proc.returncode != 0:
        err = proc.stderr if not binary else proc.stderr.decode(errors="replace")
        print(f"api-check: {' '.join(args)} failed:\n{err}", file=sys.stderr)
        sys.exit(2)
    return proc.stdout


def latest_release() -> str | None:
    tags = run(["git", "tag", "--list", "v1.*"], ROOT).split()
    releases = sorted((tuple(int(n) for n in m.groups()), t) for t in tags if (m := RELEASE.match(t)))
    return releases[-1][1] if releases else None


def uncovered(listing: str) -> list[str]:
    """The root-module packages in a `go list -f {{.ImportPath}} ./...`
    listing that no subtree of SUBTREES holds and that are not internal: the
    ones apidiff would never compare."""
    out = []
    for path in listing.split():
        rel = path.removeprefix(MODULE + "/")
        if rel.startswith(("internal/", *(s + "/" for s in SUBTREES))) or rel in SUBTREES:
            continue
        out.append(rel)
    return out


def compare(old_tree: Path, new_tree: Path, work: Path) -> str:
    """apidiff's incompatible changes from old_tree to new_tree, one run per
    subtree, the reports one after the other."""
    reports = []
    for subtree in SUBTREES:
        old, new = work / f"old-{subtree}.x", work / f"new-{subtree}.x"
        run(["go", "run", APIDIFF, "-m", "-w", str(old), f"{MODULE}/{subtree}"], old_tree)
        run(["go", "run", APIDIFF, "-m", "-w", str(new), f"{MODULE}/{subtree}"], new_tree)
        reports.append(run(["go", "run", APIDIFF, "-m", "-incompatible", str(old), str(new)], ROOT))
    return "\n".join(reports)


def main() -> int:
    tag = latest_release()
    if tag is None:
        print("api-check: no v1 release tag yet, so nothing to compare; it fails on an incompatible change from v1.0.0 on")
        return 0
    stray = uncovered(run(["go", "list", "-f", "{{.ImportPath}}", "./..."], ROOT))
    if stray:
        print(f"api-check: package(s) outside {', '.join(SUBTREES)} that no apidiff run compares: {', '.join(stray)}",
              file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as tmp:
        old_tree = Path(tmp) / "old"
        archive = run(["git", "archive", tag], ROOT, binary=True)
        with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
            tar.extractall(old_tree, filter="data")
        out = compare(old_tree, ROOT, Path(tmp))
    changes = [line[2:] for line in out.splitlines() if line.startswith("- ")]
    breaking = [c for c in changes if not EXPERIMENTAL.match(c)]
    for c in changes:
        print(f"api-check: incompatible{' (experimental, allowed)' if EXPERIMENTAL.match(c) else ''}: {c}")
    print(f"api-check: against {tag}, {len(breaking)} incompatible change(s) outside llmprovider/x/")
    return 1 if breaking else 0


if __name__ == "__main__":
    sys.exit(main())
