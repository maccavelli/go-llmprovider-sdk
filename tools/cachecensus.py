#!/usr/bin/env python3
"""cachecensus: the tests that leave a loopback URL in the process's catalog caches.

Defined by docs/decisions/0058-MADR-test-cache-isolation-for-reused-ports.md
D5, and written in 0058-PLAN Phase 1. httptest reuses ports, so an entry a
test leaves in the process's metadata, listing, Ollama show or probe cache
under its server's URL can be read by a later test whose server gets the
same port.

The working tree is cloned with tools/_common.scratch_clone. Five places are
instrumented in the clone, each anchor asserted to occur once in the tree
first: the metadata fetch, the two listing writes, the Ollama show reply
(0058-MADR's amendment of 2026-10-23) and the probe run. Each
prints "CENSUS <kind> <key>" on stderr when the store is the process's and
the key holds 127.0.0.1. `go test -count=1 -v -p 1 -parallel 1` then runs the
packages, and each line is given to the package and top-level test that the
log's order puts it in. A line from a fetch that outlives its test goes to the
test after it.

Usage:
  tools/cachecensus.py [--src DIR] [--out DIR] [--json] [PACKAGE...]

PACKAGE defaults to ./... . The log is kept in the output directory.

Exit codes: 0 the tests passed, 1 a test failed, 2 tool error (an anchor that
does not occur once, or a clone or a load wait that failed).
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from collections.abc import Sequence
from pathlib import Path

from _common import ROOT, load_limit, out_dir, scratch_clone, wait_load

# (file, anchor, replacement): the anchor must occur once in the tree.
INSTRUMENTS: list[tuple[str, str, str]] = [
    ("llmprovider/catalog/model_metadata.go",
     "\tfetch := &metadataFetch{done: make(chan struct{})}\n\tc.metadataFetches[url] = fetch\n",
     "\tfetch := &metadataFetch{done: make(chan struct{})}\n\tc.metadataFetches[url] = fetch\n"
     "\tcensus(\"metadata\", c, url)\n"),
    ("llmprovider/catalog/listing_cache.go",
     "func (c *store) storeListing(key listingKey, models map[string]ModelInfo, merge bool) {\n",
     "func (c *store) storeListing(key listingKey, models map[string]ModelInfo, merge bool) {\n"
     "\tcensus(\"listing\", c, key.baseURL)\n"),
    ("llmprovider/catalog/listing_cache.go",
     "\tif len(uniqueIDs(f.usable)) == 0 {\n\t\treturn now\n\t}\n",
     "\tif len(uniqueIDs(f.usable)) == 0 {\n\t\treturn now\n\t}\n\tcensus(\"listed\", c, key.baseURL)\n"),
    ("llmprovider/catalog/listing_cache.go",
     "func (c *store) storeShow(key listingKey, name string, info ModelInfo, digest string) {\n",
     "func (c *store) storeShow(key listingKey, name string, info ModelInfo, digest string) {\n"
     "\tcensus(\"shows\", c, key.baseURL)\n"),
    ("llmprovider/internal/transport/probe.go",
     "\tgenerate func(ctx context.Context, modelID string) (string, error),\n) []string {\n",
     "\tgenerate func(ctx context.Context, modelID string) (string, error),\n) []string {\n"
     "\tcensus(s, key)\n"),
]

# New files in the clone, holding the census functions the anchors call.
HELPERS: dict[str, str] = {
    "llmprovider/catalog/zz_cachecensus.go": """package catalog

import (
\t"fmt"
\t"os"
\t"strings"
)

func census(kind string, s *store, key string) {
\tif s == processStore && strings.Contains(key, "127.0.0.1") {
\t\tfmt.Fprintf(os.Stderr, "CENSUS %s %s\\n", kind, key)
\t}
}
""",
    "llmprovider/internal/transport/zz_cachecensus.go": """package transport

import (
\t"fmt"
\t"os"
\t"strings"
)

func census(s *ProbeStore, key ProbeKey) {
\tif (s == nil || s == defaultProbes) && strings.Contains(key.BaseURL, "127.0.0.1") {
\t\tfmt.Fprintf(os.Stderr, "CENSUS probes %s\\n", key.BaseURL)
\t}
}
""",
}

RUN_LINE = re.compile(r"^=== (?:RUN|CONT)\s+(\S+)")
PACKAGE_LINE = re.compile(r"^(?:ok|FAIL)\s+(\S+)")
CENSUS_LINE = re.compile(r"^CENSUS (\S+) (\S+)")


def check_anchors(src: Path, instruments: Sequence[tuple[str, str, str]]) -> list[str]:
    """The anchors that do not occur exactly once in src, each named."""
    bad = []
    for rel, old, _ in instruments:
        path = src / rel
        count = path.read_text(encoding="utf-8").count(old) if path.is_file() else 0
        if count != 1:
            bad.append(f"{rel}: anchor found {count} times: {old.splitlines()[0].strip()!r}")
    return bad


def instrument(clone: Path, instruments: Sequence[tuple[str, str, str]]) -> None:
    for rel, old, new in instruments:
        path = clone / rel
        text = path.read_text(encoding="utf-8")
        assert text.count(old) == 1, rel
        path.write_text(text.replace(old, new), encoding="utf-8")
    for rel, text in HELPERS.items():
        (clone / rel).write_text(text, encoding="utf-8")


def attribute(lines: Sequence[str], module: str) -> dict[str, dict[str, list[str]]]:
    """package -> top-level test -> the kinds it wrote, from a `go test -v -p 1` log.

    A line before any test of its package goes to "?".
    """
    found: dict[str, dict[str, set[str]]] = {}
    pending: list[tuple[str, str]] = []
    current = "?"
    for line in lines:
        if m := RUN_LINE.match(line):
            current = m.group(1).split("/")[0]
        elif m := CENSUS_LINE.match(line):
            pending.append((current, m.group(1)))
        elif m := PACKAGE_LINE.match(line):
            name = m.group(1).removeprefix(module + "/")
            for test, kind in pending:
                found.setdefault(name, {}).setdefault(test, set()).add(kind)
            pending, current = [], "?"
    return {p: {t: sorted(k) for t, k in sorted(tests.items())} for p, tests in sorted(found.items())}


def module_path(src: Path) -> str:
    for line in (src / "go.mod").read_text(encoding="utf-8").splitlines():
        if line.startswith("module "):
            return line.split()[1]
    raise ValueError("go.mod names no module")


def main(argv: list[str] | None = None,
         instruments: Sequence[tuple[str, str, str]] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("packages", nargs="*", default=["./..."])
    parser.add_argument("--src", type=Path, default=ROOT, help="the tree to clone (default: this one)")
    parser.add_argument("--out", type=Path, help="the output directory (default: a temporary one)")
    parser.add_argument("--json", action="store_true", help="print the census as JSON")
    args = parser.parse_args(argv)
    instruments = INSTRUMENTS if instruments is None else instruments
    bad = check_anchors(args.src, instruments)
    if bad:
        for line in bad:
            print(f"cachecensus: {line}", file=sys.stderr)
        return 2
    out = args.out or out_dir("cachecensus")
    out.mkdir(parents=True, exist_ok=True)
    clone = out / "clone"
    try:
        scratch_clone(args.src, clone)
    except Exception as err:  # noqa: BLE001 - any failure to clone is a tool error
        print(f"cachecensus: clone: {err}", file=sys.stderr)
        return 2
    instrument(clone, instruments)
    if not wait_load(load_limit(), 1800):
        print("cachecensus: the load average stayed at or above the CPU count", file=sys.stderr)
        return 2
    # One stream, so that each CENSUS line stays between the lines of its test.
    result = subprocess.run(["go", "test", "-count=1", "-v", "-p", "1", "-parallel", "1", *args.packages],
                            cwd=clone, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    log = out / "go-test.log"
    log.write_text(result.stdout, encoding="utf-8")
    census = attribute(result.stdout.splitlines(), module_path(args.src))
    if args.json:
        print(json.dumps(census, indent=1))
    else:
        for package, tests in census.items():
            print(f"{package}: {len(tests)} test(s)")
            for test, kinds in tests.items():
                print(f"    {test} {','.join(kinds)}")
    total = sum(len(t) for t in census.values())
    print(f"cachecensus: {total} test(s) in {len(census)} package(s) wrote a loopback URL "
          f"into the process's caches; go test exit {result.returncode}; log {log}", file=sys.stderr)
    return 0 if result.returncode == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
