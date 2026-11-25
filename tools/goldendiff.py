#!/usr/bin/env python3
"""goldendiff: each G-wire golden differs from a revision only as the output-limit work allows.

Defined by docs/decisions/0088-MADR-default-output-limit-from-model-facts.md
ST100, and moved here from a phase's scratch script by
docs/decisions/0089-PLAN-model-output-limit-discovery.md (the merged phase's
layer D). The goldens regenerated with -update may differ from REV only:

* in a request body's output-limit field (max_tokens, max_output_tokens,
  generation_config.max_output_tokens, generationConfig.maxOutputTokens):
  removed where it was 8192, or its 8192 changed to another positive number;
* with --stream, a plain request becoming the stream form a plain Generate
  takes when the SDK chose its limit or left it out (0088-MADR ST88): the
  body's "stream": true and "stream_options": {"include_usage": true} added,
  the header Accept: text/event-stream added, and a generateContent path
  becoming :streamGenerateContent with the query alt=sse. Nothing else of a
  request, and nothing of a result, may change.
* with --ranked-listings, the result of exactly the five listing goldens
  that 0093-MADR D12 (e) regenerated with the extract fixture handed in, now
  ranked by that document (MH58): claude/listing.json and
  opencode-zen-{responses,messages,google,chat}/listing.json. A change to
  their requests, or to any other golden's result, is still a finding.

A golden absent at REV is new: only an explicit-8192.json is allowed, each of
its POST bodies carrying 8192 in an output-limit field or none (the ChatGPT
routes), and none in the stream form, since a request that names its own limit
stays plain (ST88e), unless the case's text.json at REV was already streamed:
a route that streams every request, as the ChatGPT backend does. A golden at
REV missing in the tree is a finding.

Usage:
  tools/goldendiff.py --rev REV [--stream] [--ranked-listings] [--repo DIR] [--root DIR]

--repo is the git repository REV is read from (this one by default); --root
the tree the goldens are read from (--repo by default).

Exit codes: 0 clean, 1 a finding, 2 tool error.
"""
from __future__ import annotations

import argparse
import json
import subprocess
import sys
from collections import Counter
from pathlib import Path, PurePosixPath

from _common import ROOT

LIMIT_KEYS = {"max_tokens", "max_output_tokens", "maxOutputTokens"}
GLOB = "llmprovider/providers/*/testdata/wire/*/*.json"
EXPLICIT = "explicit-8192.json"
LIMIT_VALUE = 8192

# ST88's stream form, by the request's own field: each added where it was absent.
STREAM_BODY = {"stream": True, "stream_options": {"include_usage": True}}
STREAM_ACCEPT = "text/event-stream"
STREAM_QUERY = {"alt": ["sse"]}
PLAIN_PATH, STREAM_PATH = ":generateContent", ":streamGenerateContent"

# 0093-MADR D12 (e), MH58: the listing goldens ranked by the extract fixture, by exact path.
RANKED_LISTINGS = frozenset({
    "llmprovider/providers/claude/testdata/wire/claude/listing.json",
    "llmprovider/providers/opencode/testdata/wire/opencode-zen-responses/listing.json",
    "llmprovider/providers/opencode/testdata/wire/opencode-zen-messages/listing.json",
    "llmprovider/providers/opencode/testdata/wire/opencode-zen-google/listing.json",
    "llmprovider/providers/opencode/testdata/wire/opencode-zen-chat/listing.json",
})


class _Missing:
    def __repr__(self) -> str:
        return "<absent>"


MISSING = _Missing()


def diffs(a, b, path=()):
    """Each difference between a and b as (path, before, after); MISSING marks an absent key."""
    if isinstance(a, dict) and isinstance(b, dict):
        for k in sorted(set(a) | set(b)):
            yield from diffs(a.get(k, MISSING), b.get(k, MISSING), path + (k,))
    elif isinstance(a, list) and isinstance(b, list) and len(a) == len(b):
        for i, (x, y) in enumerate(zip(a, b)):
            yield from diffs(x, y, path + (i,))
    elif a != b:
        yield path, a, b


def limit_change(path, before, after) -> bool:
    """A request body's output-limit field, from 8192 to none or another positive number."""
    if len(path) < 4 or path[0] != "requests" or path[2] != "body" or path[-1] not in LIMIT_KEYS:
        return False
    if before != LIMIT_VALUE:
        return False
    return after is MISSING or (isinstance(after, int) and not isinstance(after, bool) and after > 0)


def stream_change(path, before, after) -> bool:
    """One part of ST88's stream form, added to a plain request."""
    if len(path) < 3 or path[0] != "requests":
        return False
    field = path[2:]
    if len(field) == 2 and field[0] == "body" and field[1] in STREAM_BODY:
        return before is MISSING and after == STREAM_BODY[field[1]] and type(after) is type(STREAM_BODY[field[1]])
    if field == ("header", "Accept"):
        return before is MISSING and after == STREAM_ACCEPT
    if field == ("query",):
        return before is MISSING and after == STREAM_QUERY
    if field == ("path",):
        return (isinstance(before, str) and isinstance(after, str) and before.endswith(PLAIN_PATH)
                and after == before[:-len(PLAIN_PATH)] + STREAM_PATH)
    return False


def ranked_listing_change(rel: str, path) -> bool:
    """A change to the result of one of the five listing goldens of 0093-MADR D12 (e)."""
    return rel in RANKED_LISTINGS and len(path) >= 1 and path[0] == "result"


def streamed(request: dict) -> bool:
    """Whether a recorded request carries any part of the stream form."""
    body = request.get("body")
    return ((isinstance(body, dict) and any(k in body for k in STREAM_BODY))
            or str(request.get("path", "")).endswith(STREAM_PATH))


def limits_in(v):
    """Every output-limit field's value in v."""
    if isinstance(v, dict):
        for k, x in v.items():
            if k in LIMIT_KEYS:
                yield x
            else:
                yield from limits_in(x)
    elif isinstance(v, list):
        for x in v:
            yield from limits_in(x)


def at_rev(repo: Path, rev: str, rel: str):
    out = subprocess.run(["git", "-C", str(repo), "show", f"{rev}:{rel}"], capture_output=True, text=True, check=False)
    return json.loads(out.stdout) if out.returncode == 0 else None


def always_streams(repo: Path, rev: str, rel: str) -> bool:
    """Whether the case's text golden at rev already streamed: a route that streams every request (the ChatGPT
    backend), where a request that names its limit is streamed too."""
    text = at_rev(repo, rev, str(PurePosixPath(rel).with_name("text.json")))
    return text is not None and any(streamed(r) for r in text.get("requests", []) if r.get("method") == "POST")


def check_new(rel: str, doc: dict, plain_case: bool) -> list[str]:
    """The findings of a golden absent at the revision; plain_case is false on a route that always streams."""
    if PurePosixPath(rel).name != EXPLICIT:
        return [f"{rel}: new, and not an {EXPLICIT} golden"]
    findings = []
    for i, req in enumerate(doc.get("requests", [])):
        if req.get("method") != "POST":
            continue
        if any(v != LIMIT_VALUE for v in limits_in(req.get("body"))):
            findings.append(f"{rel}: request {i} carries {list(limits_in(req.get('body')))}; want {LIMIT_VALUE} or none")
        if plain_case and streamed(req):
            findings.append(f"{rel}: request {i} is in the stream form; a request that names its limit stays plain")
    return findings


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--rev", required=True, help="the revision the goldens are compared with")
    ap.add_argument("--stream", action="store_true", help="allow ST88's stream form (0088-MADR ST88)")
    ap.add_argument("--ranked-listings", action="store_true",
                    help="allow a change to the result of the five listing goldens of 0093-MADR D12 (e) (MH58)")
    ap.add_argument("--repo", type=Path, default=ROOT, help="the repository REV is read from")
    ap.add_argument("--root", type=Path, help="the tree the goldens are read from; --repo by default")
    args = ap.parse_args(argv)
    root = args.root or args.repo
    listed = subprocess.run(["git", "-C", str(args.repo), "ls-tree", "-r", "--name-only", args.rev,
                             "llmprovider/providers"], capture_output=True, text=True, check=False)
    if listed.returncode != 0:
        print(f"goldendiff: git ls-tree {args.rev}: {listed.stderr.strip()}", file=sys.stderr)
        return 2
    base = {p for p in listed.stdout.split() if PurePosixPath(p).match(GLOB)}
    tree = {p.relative_to(root).as_posix() for p in root.glob(GLOB)}
    allowed = (limit_change, stream_change) if args.stream else (limit_change,)
    findings, counts = [], Counter()
    for rel in sorted(base - tree):
        findings.append(f"{rel}: in {args.rev}, missing")
    for rel in sorted(tree):
        provider = rel.split("/")[2]
        try:
            new = json.loads((root / rel).read_text(encoding="utf-8"))
            old = at_rev(args.repo, args.rev, rel) if rel in base else None
        except ValueError as err:
            print(f"goldendiff: {rel}: {err}", file=sys.stderr)
            return 2
        if old is None:
            counts[(provider, "new")] += 1
            findings.extend(check_new(rel, new, not always_streams(args.repo, args.rev, rel)))
            continue
        found = list(diffs(old, new))
        if not found:
            counts[(provider, "unchanged")] += 1
            continue
        counts[(provider, "changed")] += 1
        for path, before, after in found:
            if args.ranked_listings and ranked_listing_change(rel, path):
                continue
            if not any(ok(path, before, after) for ok in allowed):
                findings.append(f"{rel}: {'/'.join(map(str, path))}: {before!r} -> {after!r}")
    for provider in sorted({p for p, _ in counts}):
        print(f"{provider}: changed {counts[(provider, 'changed')]}, unchanged {counts[(provider, 'unchanged')]}, "
              f"new {counts[(provider, 'new')]}")
    total = Counter()
    for (_, kind), n in counts.items():
        total[kind] += n
    print(f"total: changed {total['changed']}, unchanged {total['unchanged']}, new {total['new']}")
    for f in findings:
        print("FINDING", f)
    print(f"goldendiff: {len(findings)} finding(s)")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
