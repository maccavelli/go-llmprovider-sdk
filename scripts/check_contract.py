#!/usr/bin/env python3
"""contract-check: the nested module authcontract keeps to its own rules.

Defined by docs/decisions/0073-MADR-publish-an-auth-contract-module.md D7 and
D8; written in 0073-PLAN Phase 2. The module is data only and versions on its
own, under `authcontract/v*` tags, so it is held to a stricter gate than the
root's. It fails on:

  * a requirement in authcontract/go.mod (`go mod edit -json`), or a package
    in `go list -deps -test ./...` that is neither standard nor under the
    module's path;
  * `go vet ./...` or `go test -race -count=1 ./...` failing in the module;
  * a package with statements under 80 % coverage (R47; coverage-check reads
    the root module only). No package is exempt;
  * an incompatible API change against the latest `authcontract/v*` tag, by
    apidiff at check_api.py's pin: any change fails, though the module is at
    v0, where Go's own rules would allow it;
  * a JSON break against the schema at that tag: a `$defs` entry, a property
    or an enum value removed, a type changed, or a property newly required.
    This catches a changed JSON name, which apidiff may not report.

Before the first `authcontract/v*` tag there is nothing to compare: it says so,
the two comparisons pass, and the other checks still run.

Usage:
  scripts/check_contract.py

Exit codes: 0 clean, 1 a check failed, 2 tool error.
"""
from __future__ import annotations

import io
import json
import os
import re
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path

from check_api import APIDIFF
from check_coverage import COVER

ROOT = Path(__file__).resolve().parent.parent
MODULE = "github.com/maccavelli/go-llmprovider-sdk/authcontract"
DIR = "authcontract"
SCHEMA = f"{DIR}/authcontract.schema.json"
TAG = re.compile(r"^authcontract/v(\d+)\.(\d+)\.(\d+)$")
# A test build names its packages "path [path.test]", its external test
# package "path_test" and its main package "path.test".
TEST_VARIANT = re.compile(r" \[[^\]]*\]$")
FLOOR = 80.0
TAIL = 40


def fail(message: str) -> None:
    print(f"contract-check: {message}", file=sys.stderr)
    sys.exit(2)


def run(args: list[str], cwd: Path = ROOT, binary: bool = False) -> subprocess.CompletedProcess:
    """args in cwd, never through a shell. GOWORK=off, so that a developer's go.work
    cannot supply what the module's own go.mod does not."""
    env = {**os.environ, "GOWORK": "off"}
    try:
        return subprocess.run(args, cwd=cwd, env=env, capture_output=True, text=not binary, check=False)
    except FileNotFoundError as err:
        fail(f"cannot run {args[0]}: {err}")
        raise


def go(*args: str) -> subprocess.CompletedProcess:
    return run(["go", "-C", DIR, *args])


def tail(text: str) -> str:
    lines = text.rstrip().splitlines()
    shown = lines[-TAIL:]
    return "\n".join(("    " + line) for line in ([f"... ({len(lines) - TAIL} earlier line(s) left out)"] if len(lines) > TAIL else []) + shown)


def is_own(path: str) -> bool:
    return path == MODULE or path.startswith(MODULE + "/")


def requirements() -> list[str]:
    proc = go("mod", "edit", "-json")
    if proc.returncode != 0:
        return [f"go mod edit -json failed:\n{tail(proc.stdout + proc.stderr)}"]
    try:
        required = json.loads(proc.stdout).get("Require") or []
    except ValueError as err:
        return [f"go mod edit -json printed no JSON: {err}"]
    return [f"{DIR}/go.mod requires {r['Path']} {r['Version']}; the module requires nothing" for r in required]


def dependencies() -> list[str]:
    proc = go("list", "-e", "-deps", "-test", "-f", "{{.ImportPath}}\t{{.Standard}}", "./...")
    if proc.returncode != 0:
        return [f"go list -deps -test failed:\n{tail(proc.stdout + proc.stderr)}"]
    outside: set[str] = set()
    for line in proc.stdout.splitlines():
        path, _, standard = line.partition("\t")
        path = TEST_VARIANT.sub("", path).removesuffix(".test").removesuffix("_test")
        if standard != "true" and not is_own(path):
            outside.add(path)
    return [f"{DIR} depends on {p}, which is neither standard nor under {MODULE}" for p in sorted(outside)]


def vet() -> list[str]:
    proc = go("vet", "./...")
    return [] if proc.returncode == 0 else [f"go vet failed:\n{tail(proc.stdout + proc.stderr)}"]


def tests_and_coverage() -> list[str]:
    proc = go("test", "-race", "-count=1", "-cover", "./...")
    text = proc.stdout + proc.stderr
    problems = [] if proc.returncode == 0 else [f"go test -race failed:\n{tail(text)}"]
    for line in text.splitlines():
        m = COVER.match(line)
        if m is None:
            continue
        pkg = DIR + m.group(1).removeprefix(MODULE)
        if m.group(2) is None:
            print(f"contract-check: {pkg}: no statements")
            continue
        got = float(m.group(2))
        print(f"contract-check: {pkg}: {got:.1f}% against {FLOOR:.1f}%")
        if got < FLOOR:
            problems.append(f"{pkg} is {got:.1f}%, below the {FLOOR:.1f}% floor")
    return problems


def latest_tag() -> str | None:
    proc = run(["git", "tag", "--list", f"{DIR}/v*"])
    if proc.returncode != 0:
        fail(f"git tag --list failed:\n{proc.stderr}")
    found = sorted((tuple(int(n) for n in m.groups()), t) for t in proc.stdout.split() if (m := TAG.match(t)))
    return found[-1][1] if found else None


def apidiff(tag: str) -> list[str]:
    """Any incompatible change against tag, by apidiff -m, as check_api.py runs it for the root."""
    with tempfile.TemporaryDirectory() as tmp:
        old_tree, old, new = Path(tmp) / "old", Path(tmp) / "old.x", Path(tmp) / "new.x"
        archive = run(["git", "archive", tag, DIR], binary=True)
        if archive.returncode != 0:
            fail(f"git archive {tag} {DIR} failed:\n{archive.stderr.decode(errors='replace')}")
        with tarfile.open(fileobj=io.BytesIO(archive.stdout)) as tar:
            tar.extractall(old_tree, filter="data")
        for args, cwd in ((["-m", "-w", str(old), MODULE], old_tree / DIR), (["-m", "-w", str(new), MODULE], ROOT / DIR),
                          (["-m", "-incompatible", str(old), str(new)], ROOT / DIR)):
            proc = run(["go", "run", APIDIFF, *args], cwd=cwd)
            if proc.returncode != 0:
                fail(f"apidiff {' '.join(args[:2])} failed:\n{proc.stderr}")
        changes = [line[2:] for line in proc.stdout.splitlines() if line.startswith("- ")]
    return [f"apidiff, against {tag}: incompatible change: {c}" for c in changes]


def schema_at(tag: str) -> str | None:
    """The committed schema as tag holds it, or None when the tag holds none."""
    if run(["git", "cat-file", "-e", f"{tag}:{SCHEMA}"]).returncode != 0:
        return None
    proc = run(["git", "show", f"{tag}:{SCHEMA}"])
    if proc.returncode != 0:
        fail(f"git show {tag}:{SCHEMA} failed:\n{proc.stderr}")
    return proc.stdout


def compare(where: str, was: dict, now: dict, problems: list[str]) -> None:
    """Append what a reader of the schema was is no longer given by now: a type
    changed, an enum value or a property removed, a property newly required."""
    if was.get("type") != now.get("type") or was.get("$ref") != now.get("$ref"):
        problems.append(f"{where}: type changed from {was.get('type', was.get('$ref'))} "
                        f"to {now.get('type', now.get('$ref'))}")
    problems.extend(f"enum value {v} removed from {where}" for v in was.get("enum", []) if v not in now.get("enum", []))
    now_props = now.get("properties", {})
    for prop, was_prop in was.get("properties", {}).items():
        if prop not in now_props:
            problems.append(f"property {prop} removed from {where}")
        else:
            compare(f"{where}.{prop}", was_prop, now_props[prop], problems)
    problems.extend(f"property {r} newly required in {where}"
                    for r in now.get("required", []) if r not in was.get("required", []))
    for key in ("items", "additionalProperties"):
        if isinstance(was.get(key), dict):
            compare(f"{where}[{key}]", was[key], now[key] if isinstance(now.get(key), dict) else {}, problems)


def schema_breaks(was: dict, now: dict) -> list[str]:
    problems: list[str] = []
    now_defs = now.get("$defs", {})
    for name, entry in was.get("$defs", {}).items():
        if name not in now_defs:
            problems.append(f"$defs entry {name} removed")
        else:
            compare(name, entry, now_defs[name], problems)
    return problems


def schema(tag: str) -> list[str]:
    was = schema_at(tag)
    path = ROOT / SCHEMA
    if was is None:
        print(f"contract-check: {tag} holds no {SCHEMA}, so there is no schema to compare")
        return []
    if not path.is_file():
        return [f"{SCHEMA} is gone; {tag} holds it"]
    try:
        old, new = json.loads(was), json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError) as err:
        return [f"{SCHEMA}, against {tag}: cannot read it: {err}"]
    return [f"schema, against {tag}: {p}" for p in schema_breaks(old, new)]


def main() -> int:
    problems: list[str] = []
    for check in (requirements, dependencies, vet, tests_and_coverage):
        problems += check()
    tag = latest_tag()
    if tag is None:
        print(f"contract-check: no {DIR}/v* tag yet, so nothing to compare; the API and the schema are held from the first tag on")
    else:
        problems += apidiff(tag) + schema(tag)
    for p in problems:
        print(f"contract-check: {p}")
    print(f"contract-check: {DIR}{f', against {tag},' if tag else ','} {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
