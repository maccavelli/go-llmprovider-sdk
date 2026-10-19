#!/usr/bin/env python3
"""docs-check: the examples, the concurrency docs and the error docs.

Defined by docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md D8, for
the standards guide's R50 to R52; written in 0048-PLAN Phase 6. It runs the
API audit, `go run ./llmprovider/internal/cmd/sdkaudit -check .`, which
prints each violation as "path:line Rnn ...", and returns its exit code:

  * R50: a public package with no Example that has an // Output: comment, or
    a package comment of llmprovider, providers or wizard with no indented
    code block;
  * R51: an exported type whose doc does not say whether it is safe for
    concurrent use;
  * R52: an exported function or method returning an error whose doc names
    no error.

Usage:
  scripts/check_docs.py

Exit codes: 0 clean, 1 a violation, 2 tool error (go missing, or the audit
could not run).
"""
from __future__ import annotations

import re
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
AUDIT = "./llmprovider/internal/cmd/sdkaudit"


def main() -> int:
    if shutil.which("go") is None:
        print("docs-check: go is not on PATH", file=sys.stderr)
        return 2
    proc = subprocess.run(["go", "run", AUDIT, "-check", "."], cwd=ROOT, capture_output=True, text=True, check=False)
    sys.stdout.write(proc.stdout)
    # go run exits 1 for any non-zero exit of the program, and reports the
    # program's own as "exit status N"; a build failure has no such line.
    status = 0 if proc.returncode == 0 else 2
    rest = []
    for line in proc.stderr.splitlines():
        if m := re.fullmatch(r"exit status (\d+)", line):
            status = int(m.group(1))
        else:
            rest.append(line + "\n")
    sys.stderr.write("".join(rest))
    if status not in (0, 1):
        print(f"docs-check: the audit exited {status}", file=sys.stderr)
        return 2
    return status


if __name__ == "__main__":
    sys.exit(main())
