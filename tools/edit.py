#!/usr/bin/env python3
"""edit: exact replacements, appends and new files from a JSON spec, each asserted, all applied or none.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1, and
extended by 0047-MADR D2. The spec is a list of objects, each one of:

  {"file", "old", "new"}  "old" must occur exactly "count" times (1 by
                          default) in the file as the earlier entries leave
                          it, and is replaced by "new";
  {"file", "append"}      the text is added at the file's end, after a
                          newline if the file does not end with one;
  {"file", "create"}      a new file with the text; the file must not exist.

Any text field may instead be "<field>_file": the text of that file, relative
to the spec's own directory, so multi-line text needs no JSON escaping.
Every entry is checked before any file is written, so a spec that fails
changes nothing.

Usage:
  tools/edit.py [--root DIR] SPEC.json

A relative "file" is read from the root.

Exit codes: 0 applied, 1 an entry did not match or would create a file that
exists, 2 tool error or a bad spec.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from _common import ROOT, has_text, text_field


def plan(root: Path, spec: list[dict], base: Path) -> tuple[dict[Path, str], list[str]]:
    """The files' new text, and the entries that did not match. base is the
    directory a "<field>_file" is read from."""
    texts: dict[Path, str] = {}
    problems: list[str] = []
    for entry in spec:
        name = entry["file"]
        shapes = [s for s in ("old", "append", "create") if has_text(entry, s)]
        if len(shapes) != 1:
            raise ValueError(f"{name}: an entry is one of old/new, append or create, not {shapes or 'none'}")
        path = Path(name) if Path(name).is_absolute() else root / name
        if shapes == ["create"]:
            if path in texts or path.exists():
                problems.append(f"{name}: exists; create wants a new file")
                continue
            texts[path] = text_field(entry, "create", base)
            continue
        if path not in texts:
            texts[path] = path.read_text(encoding="utf-8")
        if shapes == ["append"]:
            current = texts[path]
            joint = "" if current == "" or current.endswith("\n") else "\n"
            texts[path] = current + joint + text_field(entry, "append", base)
            continue
        old, new = text_field(entry, "old", base), text_field(entry, "new", base)
        want = int(entry.get("count", 1))
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
        texts, problems = plan(args.root.resolve(), spec, args.spec.resolve().parent)
    except (OSError, ValueError, KeyError, TypeError) as err:
        print(f"edit: {err}", file=sys.stderr)
        return 2
    if problems:
        for p in problems:
            print(p)
        print(f"edit: {len(problems)} entry(ies) did not match; nothing written")
        return 1
    for path, text in texts.items():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
    print(f"edit: {len(spec)} entry(ies) applied to {len(texts)} file(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
