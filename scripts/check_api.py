#!/usr/bin/env python3
"""api-check: no incompatible API change against the latest v1 release.

Defined by docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md D13
and the standards guide's R48; written in 0015-PLAN S12. It compares the
module's exported API with the latest `v1.X.Y` tag (release candidates do not
count) using apidiff at a pinned version, run with `go run`: a tool, not a
module requirement. Internal packages are not compared, and changes under
llmprovider/x/, the experimental tree, are reported without failing.

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
RELEASE = re.compile(r"^v1\.(\d+)\.(\d+)$")
EXPERIMENTAL = "./llmprovider/x/"


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


def main() -> int:
    tag = latest_release()
    if tag is None:
        print("api-check: no v1 release tag yet, so nothing to compare; it fails on an incompatible change from v1.0.0 on")
        return 0
    with tempfile.TemporaryDirectory() as tmp:
        old_tree, old, new = Path(tmp) / "old", Path(tmp) / "old.x", Path(tmp) / "new.x"
        archive = run(["git", "archive", tag], ROOT, binary=True)
        with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
            tar.extractall(old_tree, filter="data")
        run(["go", "run", APIDIFF, "-m", "-w", str(old), MODULE], old_tree)
        run(["go", "run", APIDIFF, "-m", "-w", str(new), MODULE], ROOT)
        out = run(["go", "run", APIDIFF, "-m", "-incompatible", str(old), str(new)], ROOT)
    changes = [line[2:] for line in out.splitlines() if line.startswith("- ")]
    breaking = [c for c in changes if not c.startswith(EXPERIMENTAL)]
    for c in changes:
        print(f"api-check: incompatible{' (experimental, allowed)' if c.startswith(EXPERIMENTAL) else ''}: {c}")
    print(f"api-check: against {tag}, {len(breaking)} incompatible change(s) outside llmprovider/x/")
    return 1 if breaking else 0


if __name__ == "__main__":
    sys.exit(main())
