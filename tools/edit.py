#!/usr/bin/env python3
"""edit: exact replacements from a JSON spec, each asserted, all applied or none.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1. The spec
is a list of {"file", "old", "new"} objects, with an optional "count" (1 by
default): "old" must occur exactly that many times in the file as the earlier
entries leave it. Every entry is checked before any file is written, so a
spec that fails changes nothing.

Usage:
  tools/edit.py [--root DIR] SPEC.json

A relative "file" is read from the root.

Exit codes: 0 applied, 1 an entry did not match, 2 tool error or a bad spec.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from _common import ROOT


def plan(root: Path, spec: list[dict]) -> tuple[dict[Path, str], list[str]]:
    """The files' new text, and the entries that did not match."""
    texts: dict[Path, str] = {}
    problems: list[str] = []
    for entry in spec:
        name, old, new = entry["file"], entry["old"], entry["new"]
        want = int(entry.get("count", 1))
        path = Path(name) if Path(name).is_absolute() else root / name
        if path not in texts:
            texts[path] = path.read_text(encoding="utf-8")
        got = texts[path].count(old)
        if got != want:
            problems.append(f"{name}: {old[:60]!r} matches {got} time(s), want {want}")
            continue
        texts[path] = texts[path].replace(old, new, want)
    return texts, problems


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT, help="the repository (default: this one)")
    parser.add_argument("spec", type=Path, help="the JSON spec")
    args = parser.parse_args(argv)
    try:
        spec = json.loads(args.spec.read_text(encoding="utf-8"))
        if not isinstance(spec, list):
            raise ValueError("the spec is not a list")
        texts, problems = plan(args.root.resolve(), spec)
    except (OSError, ValueError, KeyError, TypeError) as err:
        print(f"edit: {err}", file=sys.stderr)
        return 2
    if problems:
        for p in problems:
            print(p)
        print(f"edit: {len(problems)} entry(ies) did not match; nothing written")
        return 1
    for path, text in texts.items():
        path.write_text(text, encoding="utf-8")
    print(f"edit: {len(spec)} entry(ies) applied to {len(texts)} file(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
