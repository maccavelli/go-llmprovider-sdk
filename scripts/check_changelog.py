#!/usr/bin/env python3
"""changelog-check: CHANGELOG.md has a section for every release tag, and no released section changes.

Defined by docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md D5;
written in 0048-PLAN Phase 5. CHANGELOG.md is in the Keep a Changelog 1.1.0
form: an optional "## [Unreleased]" section first, then one
"## [vX.Y.Z] - YYYY-MM-DD" section per release, newest first; a released
heading may end " [YANKED]" (0094-MADR D5). It fails on:

  * a v* tag with no section;
  * sections out of order, by version, newest first;
  * a section whose date is not its tag's commit date
    (git log -1 --format=%cs TAG);
  * a section with no tag, unless it is newer than every tag: the release
    being prepared, which its tag will name; and any section with no tag
    marked [YANKED];
  * a heading of another form, or a version given twice;
  * a released section (one whose tag exists) whose body differs from its
    text in the first tag's file, by version at or after its own, that
    carries it; a released tag's compare link that differs from its line
    there; a second compare link for a released tag (0094-MADR D1, D2).

A tag's file carries a section when one of its level-two headings starts
"## [vX.Y.Z]", whatever follows. So v1.0.0 to v1.15.0, which no file had at
their own tags, are compared with v1.16.0's file, and v1.24.0 with v1.24.1's.
A body is the section's lines after its heading, to the next level-two
heading, or for the last section to the compare links; each line's trailing
white space and the body's leading and trailing blank lines are dropped,
and nothing else. A released section no file carries fails, unless its tag
is the newest: a tag made before its date step. [Unreleased], the preamble
and the section of the release being prepared are never compared. A needed
fix to a released section goes through ACCEPTED, below; an entry for no
release tag, or for a tag with no section, fails.

The comparison reads the tags' files with one `git cat-file --batch-check`
and one `git cat-file --batch` per file, and fetches nothing. Missing history
is a tool error, never a pass: a shallow clone, a tag or accepted commit the
clone does not hold, two or more release sections newer than every tag (the
clone lacks tags), or a tag's file that is not UTF-8; so is a working
changelog that is not UTF-8. CI checks out with fetch-depth: 0. A missing
tag is seen only from above: a lost middle tag fails as a section with no
tag (exit 1), since the gate cannot tell it from one never tagged, and a
lost newest tag reads as the release being prepared (0094-MADR RS38).

authcontract/CHANGELOG.md, the nested module's, is held to the same rules by
its own tags, `authcontract/vX.Y.Z`: a section is named for the version alone,
and a problem names the full tag, with the file's name in front
(0073-MADR D8).

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
import dataclasses
import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CHANGELOG = ROOT / "CHANGELOG.md"
# A nested module's changelog and the prefix of its tags (0073-MADR D8).
NESTED = (ROOT / "authcontract/CHANGELOG.md", "authcontract/")
UNRELEASED = "## [Unreleased]"
YANKED = " [YANKED]"
RELEASE = re.compile(r"^## \[(v(\d+)\.(\d+)\.(\d+))\] - (\d{4}-\d{2}-\d{2})( \[YANKED\])?$")
VERSION = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")
# A level-two heading and the name in its brackets: a version's section is
# carried by its version alone, whatever its date or suffix (0094-MADR D1).
SECTION = re.compile(r"^## \[([^\]]+)\]")
# A compare-link line (0094-MADR D2).
LINKDEF = re.compile(r"^\[(Unreleased|v\d+\.\d+\.\d+)\]: \S+$")
# A needed fix to a released section (0094-MADR RS13): the full tag, the
# reviewed commit whose file becomes that section's baseline, body and compare
# link, and the record that accepted it, "NNNN-MADR …". That record's PLAN
# lists `gh release edit` for the tag's Release, an owner step (RS28).
ACCEPTED: dict[str, tuple[str, str]] = {}
ACCEPTED_TAG = re.compile(r"^(?:authcontract/)?v\d+\.\d+\.\d+$")
COMMIT = re.compile(r"^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
RECORD = re.compile(r"^\d{4}-MADR\b")
# An inline link's target, and a target that is not relative: a scheme, a
# fragment alone, or a path from the host's root.
LINK = re.compile(r"\]\(([^)\s]+)\)")
ABSOLUTE = re.compile(r"^(?:[A-Za-z][A-Za-z0-9+.-]*:|#|/)")
FETCH = "CI checks out with fetch-depth: 0"


class ToolError(Exception):
    """Missing history, a tag's file that is not UTF-8, or a malformed
    ACCEPTED entry: exit 2, never a pass (0094-MADR D4)."""


@dataclasses.dataclass
class Tally:
    """What the comparison did, for the summary (0094-MADR D6)."""
    compared: int = 0
    unbased: int = 0
    accepted: list[str] = dataclasses.field(default_factory=list)


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


def bodies(lines: list[str]) -> dict[str, tuple[int, list[tuple[int, str]]]]:
    """Each level-two section's name, by its first occurrence: its heading's
    line index, and its body as (line index, text) pairs, normalised as
    0094-MADR D2 says."""
    heads = [i for i, line in enumerate(lines) if line.startswith("## ")]
    out: dict[str, tuple[int, list[tuple[int, str]]]] = {}
    for n, i in enumerate(heads):
        m = SECTION.match(lines[i])
        if m is None or m.group(1) in out:
            continue
        if n + 1 < len(heads):
            end = heads[n + 1]
        else:
            end = next((j for j in range(i + 1, len(lines)) if LINKDEF.match(lines[j])), len(lines))
        body = [(j, lines[j].rstrip()) for j in range(i + 1, end)]
        while body and not body[0][1]:
            body.pop(0)
        while body and not body[-1][1]:
            body.pop()
        out[m.group(1)] = (i, body)
    return out


def links(lines: list[str]) -> dict[str, list[tuple[int, str]]]:
    """Each compare-link label's definitions, as (line index, line)."""
    found: dict[str, list[tuple[int, str]]] = {}
    for i, line in enumerate(lines):
        if m := LINKDEF.match(line):
            found.setdefault(m.group(1), []).append((i, line))
    return found


def first_difference(was: list[str], now: list[str]) -> int | None:
    """The index of the first line where now differs from was, or None."""
    for k, (a, b) in enumerate(zip(was, now)):
        if a != b:
            return k
    return None if len(was) == len(now) else min(len(was), len(now))


def read_files(revs: list[str], path: str) -> dict[str, str | None]:
    """The file at path in each rev's commit, None where that commit has no
    such file, with one `git cat-file --batch-check` and one `git cat-file
    --batch` however many revs (0094-MADR RS16). A rev whose commit the clone
    does not hold, and a file that is not UTF-8, are tool errors (D4)."""
    revs = list(dict.fromkeys(revs))
    if not revs:
        return {}
    listed = subprocess.run(["git", "cat-file", "--batch-check"], cwd=ROOT, capture_output=True, text=True,
                            input="".join(f"{rev}^{{commit}}\n" for rev in revs), check=False)
    answers = listed.stdout.splitlines()
    if listed.returncode != 0 or len(answers) != len(revs):
        raise ToolError(f"git cat-file --batch-check failed: {listed.stderr.strip()}")
    commits: dict[str, str] = {}
    for rev, answer in zip(revs, answers):
        fields = answer.split()
        if len(fields) != 3 or fields[1] != "commit":
            raise ToolError(f"{rev}: this clone does not hold its commit; {FETCH}")
        commits[rev] = fields[0]
    read = subprocess.run(["git", "cat-file", "--batch"], cwd=ROOT, capture_output=True,
                          input="".join(f"{commits[rev]}:{path}\n" for rev in revs).encode(), check=False)
    if read.returncode != 0:
        raise ToolError(f"git cat-file --batch failed: {read.stderr.decode(errors='replace').strip()}")
    data, pos, files = read.stdout, 0, {}
    for rev in revs:
        end = data.index(b"\n", pos)
        header = data[pos:end].decode(errors="replace").split()
        pos = end + 1
        if header[-1] == "missing":
            files[rev] = None
            continue
        size = int(header[2])
        try:
            files[rev] = data[pos:pos + size].decode("utf-8")
        except UnicodeDecodeError:
            raise ToolError(f"{path} at {rev} is not UTF-8") from None
        pos += size + 1
    return files


def check_accepted() -> None:
    """Each ACCEPTED entry's form: a release tag, a full object name and a
    record (0094-MADR RS13)."""
    for tag, (commit, record) in ACCEPTED.items():
        if not ACCEPTED_TAG.match(tag) or not COMMIT.match(commit) or not RECORD.match(record):
            raise ToolError(f"ACCEPTED[{tag!r}] = ({commit!r}, {record!r}) is not a release tag, a full object "
                            "name and its record, NNNN-MADR …")


def released(lines: list[str], prefix: str, path: str, listed: list[str], tally: Tally) -> list[str]:
    """The problems of the released sections of lines, the file at path,
    whose tags are prefix + each of listed: each body and compare link
    against its baseline (0094-MADR D1, D2, D6). Only a tag of the form
    vX.Y.Z releases its version or is a baseline (RS32)."""
    accepted = {t.removeprefix(prefix): entry for t, entry in ACCEPTED.items()
                if t.startswith(prefix) and VERSION.match(t.removeprefix(prefix))}
    releases = {t: prefix + t for t in listed if VERSION.match(t)}
    order = sorted(releases, key=version)
    files = read_files([releases[t] for t in order] + [commit for commit, _ in accepted.values()], path)
    parsed = {rev: (bodies(text.splitlines()), links(text.splitlines())) for rev, text in files.items()
              if text is not None}
    now, now_links = bodies(lines), links(lines)
    newest = order[-1] if order else None

    def carries(rev: str, name: str) -> bool:
        return rev in parsed and name in parsed[rev][0]

    problems: list[str] = []
    for name, (i, body) in now.items():
        if name not in releases:
            continue
        full = prefix + name
        tagged = next((releases[t] for t in order if version(t) >= version(name) and carries(releases[t], name)),
                      None)
        if name in accepted:
            base, record = accepted[name]
            if not carries(base, name):
                problems.append(f"line {i + 1}: {full}'s section has no text at {base}, the commit ACCEPTED names")
                continue
            tally.accepted.append(f"accepted ({record}): {full} at {base}")
        elif tagged is not None:
            base = tagged
        elif name == newest:
            tally.unbased += 1
            continue
        else:
            problems.append(f"line {i + 1}: {full}'s section has no released text: no tag's file carries it")
            continue
        tally.compared += 1
        was = parsed[base][0][name][1]
        k = first_difference([text for _, text in was], [text for _, text in body])
        if k is not None:
            at = body[k][0] + 1 if k < len(body) else (body[-1][0] + 2 if body else i + 2)
            problems.append(f"line {i + 1}: {full}'s section differs from its text at {base}, first at line {at}")
        was_link = parsed[base][1].get(name)
        if was_link:
            mine = now_links.get(name)
            if not mine or mine[0][1] != was_link[0][1]:
                problems.append(f"line {(mine[0][0] if mine else i) + 1}: {full}'s compare link differs from its "
                                f"line at {base}")
    for name, defs in now_links.items():
        if name in releases and len(defs) > 1:
            problems.append(f"line {defs[1][0] + 1}: a second compare link for {prefix}{name}")
    # A stale entry is a problem, so the list cannot rot (0094-MADR RS37).
    for name in sorted(accepted, key=version):
        if name not in releases:
            problems.append(f"ACCEPTED names {prefix}{name}, which is not a release tag in this clone")
        elif name not in now:
            problems.append(f"ACCEPTED names {prefix}{name}, which has no section")
    return problems


def check(lines: list[str], prefix: str = "", label: str = "", tally: Tally | None = None) -> list[str]:
    """The problems of a changelog whose sections are named for the version
    and whose tags are prefix + version. Each problem names the full tag, and,
    when label is given, the file."""
    tally = tally if tally is not None else Tally()
    path = label or CHANGELOG.name
    found, problems = sections(lines)
    listed = [t.removeprefix(prefix) for t in git("tag", "--list", f"{prefix}v*").split() if t.startswith(prefix)]
    tags = [t for t in listed if VERSION.match(t)]
    newest = max(tags, key=version) if tags else None
    seen: dict[str, int] = {}
    ahead = 0
    for tag, date, i in found:
        if tag in seen:
            problems.append(f"line {i + 1}: {prefix}{tag} has a section already, at line {seen[tag] + 1}")
            continue
        seen[tag] = i
        if tag not in tags:
            if lines[i].endswith(YANKED):
                problems.append(f"line {i + 1}: {prefix}{tag} is marked [YANKED] and has no tag")
            if newest is not None and version(tag) <= version(newest):
                problems.append(f"line {i + 1}: {prefix}{tag} has a section and no tag")
            else:
                ahead += 1
            continue
        want = git("log", "-1", "--format=%cs", prefix + tag).strip()
        if date != want:
            problems.append(f"line {i + 1}: {prefix}{tag} is dated {date}; its tag's commit is {want}")
    # At most one section can be the release being prepared (0094-MADR D4).
    if ahead >= 2:
        raise ToolError(f"{path}: {ahead} release sections are newer than every {prefix}v* tag; one can be the "
                        f"release being prepared; this clone lacks tags ({FETCH})")
    order = [tag for tag, _, _ in found]
    if order != sorted(order, key=version, reverse=True):
        problems.append(f"sections out of order, want newest first: {', '.join(prefix + t for t in order)}")
    for tag in sorted(set(tags) - set(seen), key=version, reverse=True):
        problems.append(f"{prefix}{tag} has no section")
    problems += released(lines, prefix, path, listed, tally)
    return [f"{label}: {p}" for p in problems] if label else problems


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
    except UnicodeDecodeError as e:
        print(f"changelog-check: {CHANGELOG.name} is not UTF-8: {e.reason} at byte {e.start}", file=sys.stderr)
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
    tally = Tally()
    try:
        check_accepted()
        if git("rev-parse", "--is-shallow-repository").strip() == "true":
            raise ToolError(f"this clone is shallow, so the release tags' history may be missing; {FETCH}")
        problems = check(lines, tally=tally)
        path, prefix = NESTED
        try:
            nested = path.read_text(encoding="utf-8").splitlines()
        except OSError as e:
            print(f"changelog-check: {e}", file=sys.stderr)
            return 2
        except UnicodeDecodeError as e:
            print(f"changelog-check: {path.relative_to(ROOT)} is not UTF-8: {e.reason} at byte {e.start}",
                  file=sys.stderr)
            return 2
        problems += check(nested, prefix, str(path.relative_to(ROOT)), tally)
    except ToolError as e:
        print(f"changelog-check: {e}", file=sys.stderr)
        return 2
    lines += nested
    for line in tally.accepted:
        print(f"changelog-check: {line}")
    for p in problems:
        print(f"changelog-check: {p}")
    print(f"changelog-check: {sum(1 for line in lines if RELEASE.match(line))} release section(s), "
          f"{tally.compared} compared, {len(tally.accepted)} accepted, {tally.unbased} with no released text, "
          f"{len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
