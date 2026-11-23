#!/usr/bin/env python3
"""limitsnapshot: the embedded models.dev extract, each model's limits only, from a document given by path.

Defined by docs/decisions/0089-MADR-model-output-limit-discovery.md D3 and
DL70: the catalog embeds a limits-only extract of a public models.dev
document as the registry's offline tier, regenerated at each release. This
tool reads a document already on disk (it fetches nothing) and writes:

  {"date": "YYYY-MM-DD",
   "sections": {"<section>": {"models": {"<id>": {"limit": {...}}}}}}

keeping only the sections the SDK reads (decodeModelMetadata's nine), and of
each model only its "limit" object's "context", "input" and "output". A model
with no limit is left out.

Usage:
  tools/limitsnapshot.py DOCUMENT --date YYYY-MM-DD [--out FILE]

Exit codes: 0 written, 2 tool error (an unreadable document, a bad date).
"""
from __future__ import annotations

import argparse
import datetime
import json
import sys
from pathlib import Path

from _common import ROOT

# The sections the SDK reads: decodeModelMetadata's in
# llmprovider/catalog/model_metadata.go.
SECTIONS = ("opencode", "opencode-go", "huggingface", "togetherai", "openai", "anthropic", "google", "xai", "kilo")
LIMIT_FIELDS = ("context", "input", "output")
DEFAULT_OUT = ROOT / "llmprovider" / "catalog" / "limits_snapshot.json"


def extract(doc: dict, date: str) -> dict:
    """doc's limits, by section and model, under date."""
    sections: dict = {}
    for name in SECTIONS:
        section = doc.get(name)
        if not isinstance(section, dict) or not isinstance(section.get("models"), dict):
            continue
        models = {}
        for model_id, entry in section["models"].items():
            limit = entry.get("limit") if isinstance(entry, dict) else None
            if not isinstance(limit, dict):
                continue
            kept = {k: limit[k] for k in LIMIT_FIELDS if isinstance(limit.get(k), int) and not isinstance(limit.get(k), bool)}
            if kept:
                models[model_id] = {"limit": kept}
        if models:
            sections[name] = {"models": models}
    return {"date": date, "sections": sections}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("document", type=Path, help="a models.dev document on disk")
    parser.add_argument("--date", required=True, help="the document's date, YYYY-MM-DD")
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT, help="the extract to write")
    args = parser.parse_args(argv)
    try:
        datetime.date.fromisoformat(args.date)
    except ValueError:
        print(f"limitsnapshot: --date {args.date!r} is not YYYY-MM-DD", file=sys.stderr)
        return 2
    try:
        doc = json.loads(args.document.read_text(encoding="utf-8"))
    except (OSError, ValueError) as err:
        print(f"limitsnapshot: {args.document}: {err}", file=sys.stderr)
        return 2
    if not isinstance(doc, dict):
        print(f"limitsnapshot: {args.document}: not a models.dev document", file=sys.stderr)
        return 2
    out = extract(doc, args.date)
    args.out.write_text(json.dumps(out, sort_keys=True, indent=1) + "\n", encoding="utf-8")
    count = sum(len(s["models"]) for s in out["sections"].values())
    print(f"limitsnapshot: {count} models in {len(out['sections'])} sections, dated {args.date}, to {args.out.name}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
