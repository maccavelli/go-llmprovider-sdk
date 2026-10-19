#!/usr/bin/env python3
"""changelog-check: CHANGELOG.md has a section for every release tag.

Defined by docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md D5;
written in 0048-PLAN Phase 5. CHANGELOG.md is in the Keep a Changelog 1.1.0
form: an optional "## [Unreleased]" section first, then one
"## [vX.Y.Z] - YYYY-MM-DD" section per release, newest first. It fails on:

  * a v* tag with no section;
  * sections out of order, by version, newest first;
  * a section whose date is not its tag's commit date
    (git log -1 --format=%cs TAG);
  * a section with no tag, unless it is newer than every tag: the release
    being prepared, which its tag will name;
  * a heading of another form, or a version given twice.

--section TAG prints that section's body, the lines between its heading and
the next, for a release's notes; the release job runs it. With --repo
OWNER/NAME, or GITHUB_REPOSITORY set as CI sets it, each relative link is
printed as https://github.com/OWNER/NAME/blob/TAG/PATH, its fragment kept,
since a Release's notes are read away from the tree; an absolute link is
left as it is, and a linked file that does not exist at TAG fails it
(0048-PLAN Phase 5 step 7).

Usage:
  scripts/check_changelog.py
  scripts/check_changelog.py --section vX.Y.Z [--repo OWNER/NAME]

Exit codes: 0 clean (or the section printed), 1 a check failed (or no such
section, or a link to a file TAG lacks), 2 tool error.
"""
from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CHANGELOG = ROOT / "CHANGELOG.md"
UNRELEASED = "## [Unreleased]"
RELEASE = re.compile(r"^## \[(v(\d+)\.(\d+)\.(\d+))\] - (\d{4}-\d{2}-\d{2})$")
VERSION = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")
# An inline link's target, and a target that is not relative: a scheme, a
# fragment alone, or a path from the host's root.
LINK = re.compile(r"\]\(([^)\s]+)\)")
ABSOLUTE = re.compile(r"^(?:[A-Za-z][A-Za-z0-9+.-]*:|#|/)")


def git(*args: str) -> str:
    proc = subprocess.run(["git", *args], cwd=ROOT, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        print(f"changelog-check: git {' '.join(args)} failed:\n{proc.stderr}", file=sys.stderr)
        sys.exit(2)
    return proc.stdout


def version(tag: str) -> tuple[int, int, int]:
    m = VERSION.match(tag)
    return (int(m.group(1)), int(m.group(2)), int(m.group(3))) if m else (-1, -1, -1)


def sections(lines: list[str]) -> tuple[list[tuple[str, str, int]], list[str]]:
    """Each release section's tag, date and heading line index, and the
    problems with the headings."""
    found, problems = [], []
    for i, line in enumerate(lines):
        if not line.startswith("## "):
            continue
        if line == UNRELEASED:
            if found:
                problems.append(f"line {i + 1}: [Unreleased] comes after a release")
            continue
        m = RELEASE.match(line)
        if m is None:
            problems.append(f"line {i + 1}: {line!r} is not '## [vX.Y.Z] - YYYY-MM-DD' or {UNRELEASED!r}")
            continue
        found.append((m.group(1), m.group(5), i))
    return found, problems


def check(lines: list[str]) -> list[str]:
    found, problems = sections(lines)
    tags = [t for t in git("tag", "--list", "v*").split() if VERSION.match(t)]
    newest = max(tags, key=version) if tags else None
    seen: dict[str, int] = {}
    for tag, date, i in found:
        if tag in seen:
            problems.append(f"line {i + 1}: {tag} has a section already, at line {seen[tag] + 1}")
            continue
        seen[tag] = i
        if tag not in tags:
            if newest is not None and version(tag) <= version(newest):
                problems.append(f"line {i + 1}: {tag} has a section and no tag")
            continue
        want = git("log", "-1", "--format=%cs", tag).strip()
        if date != want:
            problems.append(f"line {i + 1}: {tag} is dated {date}; its tag's commit is {want}")
    order = [tag for tag, _, _ in found]
    if order != sorted(order, key=version, reverse=True):
        problems.append(f"sections out of order, want newest first: {', '.join(order)}")
    for tag in sorted(set(tags) - set(seen), key=version, reverse=True):
        problems.append(f"{tag} has no section")
    return problems


def section(lines: list[str], tag: str) -> list[str] | None:
    found, _ = sections(lines)
    for n, (name, _, i) in enumerate(found):
        if name != tag:
            continue
        end = found[n + 1][2] if n + 1 < len(found) else len(lines)
        body = lines[i + 1:end]
        # The last section is followed by the comparison links.
        while body and (not body[-1].strip() or body[-1].startswith("[")):
            body.pop()
        while body and not body[0].strip():
            body.pop(0)
        return body
    return None


def at_tag(body: list[str], tag: str, repo: str) -> tuple[list[str], list[str]]:
    """body with each relative link pointed at its file at tag on GitHub, and
    the links whose file tag lacks."""
    missing: list[str] = []

    def rewrite(m: re.Match[str]) -> str:
        target = m.group(1)
        if ABSOLUTE.match(target):
            return m.group(0)
        path, _, fragment = target.partition("#")
        found = subprocess.run(["git", "cat-file", "-e", f"{tag}:{path}"], cwd=ROOT, capture_output=True, check=False)
        if found.returncode != 0:
            missing.append(target)
        return f"](https://github.com/{repo}/blob/{tag}/{path}" + (f"#{fragment}" if fragment else "") + ")"

    return [LINK.sub(rewrite, line) for line in body], missing


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--section", metavar="TAG", help="print the section of TAG and exit")
    parser.add_argument("--repo", metavar="OWNER/NAME", default=os.environ.get("GITHUB_REPOSITORY", ""),
                        help="point --section's relative links at TAG in this GitHub repository")
    args = parser.parse_args(argv)
    try:
        lines = CHANGELOG.read_text(encoding="utf-8").splitlines()
    except OSError as e:
        print(f"changelog-check: {e}", file=sys.stderr)
        return 2
    if args.section is not None:
        body = section(lines, args.section)
        if body is None:
            print(f"changelog-check: CHANGELOG.md has no section for {args.section}", file=sys.stderr)
            return 1
        if args.repo:
            body, missing = at_tag(body, args.section, args.repo)
            if missing:
                for target in missing:
                    print(f"changelog-check: {args.section}'s section links {target}, which {args.section} lacks",
                          file=sys.stderr)
                return 1
        print("\n".join(body))
        return 0
    problems = check(lines)
    for p in problems:
        print(f"changelog-check: {p}")
    print(f"changelog-check: {sum(1 for line in lines if RELEASE.match(line))} release section(s), "
          f"{len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
