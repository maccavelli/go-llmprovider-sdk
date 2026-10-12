#!/usr/bin/env python3
"""consumer: build a scratch consumer of every public package, and optionally move a dependent repository.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1, for the
checks after a release tag. The consumer imports every package `go list
./...` reports outside internal/, and builds for each platform with
CGO_ENABLED=0. It requires a released VERSION through the module proxy, or,
with --replace, the working tree.

--dependent PATH copies a repository that uses this module into the work
directory, without its .git, its build output, or any *.devops.*.json,
secrets.json or .env* file. It runs its tests on its own requirement, moves it
to VERSION, and then builds, vets for three platforms, tests, tests with
-race, and lints it if it has a golangci-lint config. The repository itself
is never changed.

Usage:
  tools/consumer.py VERSION [--dependent PATH] [--platforms LIST] [--out DIR]
  tools/consumer.py --replace [--platforms host] [--out DIR]

LIST is comma-separated GOOS/GOARCH pairs, or "host"; six by default.

Exit codes: 0 every step passed, 1 a step failed, 2 tool error.
"""
from __future__ import annotations

import argparse
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

from _common import ROOT, out_dir

MODULE = "github.com/maccavelli/go-llmprovider-sdk"
PLATFORMS = "linux/amd64,linux/arm64,windows/amd64,windows/arm64,darwin/amd64,darwin/arm64"
SKIP = shutil.ignore_patterns(".git", "bin", "dist", "*.devops.*.json", "secrets.json", ".env*")


def public_packages() -> list[str]:
    """Every package outside internal/ that is not a command."""
    out = subprocess.run(["go", "list", "-f", "{{.ImportPath}} {{.Name}}", "./..."], cwd=ROOT,
                         capture_output=True, text=True, check=True).stdout
    pkgs = []
    for line in out.splitlines():
        path, name = line.split()
        if "/internal" not in path and name != "main":
            pkgs.append(path)
    return pkgs


def main_source(pkgs: list[str]) -> str:
    imports = "".join(f'\t_ "{p}"\n' for p in pkgs)
    return (f"// Command consumer links every public package of {MODULE}.\npackage main\n\n"
            f'import (\n\t"fmt"\n\n{imports})\n\nfunc main() {{ fmt.Println("consumer: {len(pkgs)} packages linked") }}\n')


def go_version() -> str:
    match = re.search(r"^go (\S+)$", (ROOT / "go.mod").read_text(encoding="utf-8"), re.M)
    if not match:
        raise ValueError("go.mod names no go version")
    return match.group(1)


class Steps:
    """Runs and reports each step, and counts the failures."""

    def __init__(self) -> None:
        self.failed = 0

    def run(self, cwd: Path, *args: str, env: dict[str, str] | None = None) -> int:
        p = subprocess.run(list(args), cwd=cwd, capture_output=True, text=True, env={**os.environ, **(env or {})})
        print(f"$ {' '.join(args)}{' ' + str(env) if env else ''}: rc={p.returncode}")
        tail = (p.stdout + p.stderr).strip()
        if tail:
            print("   " + tail[-2000:].replace("\n", "\n   "))
        if p.returncode != 0:
            self.failed += 1
        return p.returncode


def platforms(spec: str) -> list[dict[str, str] | None]:
    if spec == "host":
        return [None]
    return [{"GOOS": goos, "GOARCH": arch, "CGO_ENABLED": "0"} for goos, arch in (p.split("/") for p in spec.split(","))]


def check_consumer(work: Path, version: str | None, plats: list[dict[str, str] | None], steps: Steps) -> None:
    c = work / "consumer"
    if c.exists():
        shutil.rmtree(c)
    c.mkdir(parents=True)
    (c / "main.go").write_text(main_source(public_packages()), encoding="utf-8")
    (c / "go.mod").write_text(f"module example.com/consumer\n\ngo {go_version()}\n", encoding="utf-8")
    if version:
        steps.run(c, "go", "get", f"{MODULE}@{version}")
    else:
        steps.run(c, "go", "mod", "edit", f"-require={MODULE}@v0.0.0-00010101000000-000000000000",
                  f"-replace={MODULE}={ROOT}")
    steps.run(c, "go", "mod", "tidy")
    for env in plats:
        steps.run(c, "go", "build", "-o", os.devnull, ".", env=env)
    steps.run(c, "go", "run", ".")


def check_dependent(work: Path, src: Path, version: str, steps: Steps) -> None:
    t = work / "dependent"
    if t.exists():
        shutil.rmtree(t)
    shutil.copytree(src, t, ignore=SKIP)
    steps.run(t, "go", "test", "-count=1", "./...")  # the baseline, on its own requirement
    steps.run(t, "go", "get", f"{MODULE}@{version}")
    steps.run(t, "go", "mod", "tidy")
    steps.run(t, "go", "build", "./...")
    for goos in ("darwin", "linux", "windows"):
        steps.run(t, "go", "vet", "./...", env={"GOOS": goos, "CGO_ENABLED": "0"})
    steps.run(t, "go", "test", "-count=1", "./...")
    steps.run(t, "go", "test", "-count=1", "-race", "./...")
    for cfg in (".golangci.yml", ".golangci.yaml"):
        if (t / cfg).exists() and shutil.which("golangci-lint"):
            steps.run(t, "golangci-lint", "run", "-c", cfg, "./...")
            break


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("version", nargs="?", help="the released version, such as v1.9.0")
    parser.add_argument("--replace", action="store_true", help="build against the working tree instead")
    parser.add_argument("--dependent", type=Path, help="a repository that uses this module, to move to VERSION")
    parser.add_argument("--platforms", default=PLATFORMS, help='GOOS/GOARCH pairs, or "host"')
    parser.add_argument("--out", type=Path, default=None, help="the work directory")
    args = parser.parse_args(argv)
    if bool(args.version) == args.replace:
        parser.error("give a VERSION, or --replace, not both")
    if args.dependent and not args.version:
        parser.error("--dependent needs a VERSION")
    work = args.out or out_dir("consumer")
    steps = Steps()
    try:
        check_consumer(work, args.version, platforms(args.platforms), steps)
        if args.dependent:
            check_dependent(work, args.dependent.resolve(), args.version, steps)
    except (OSError, ValueError, subprocess.CalledProcessError) as err:
        print(f"consumer: {err}", file=sys.stderr)
        return 2
    if steps.failed:
        print(f"consumer: {steps.failed} step(s) failed")
        return 1
    print("consumer: every step passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
