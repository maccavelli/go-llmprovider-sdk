"""Shared code for the maintainer tools in tools/.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D1 and D2.
The repository root is found from this file, never from the working directory
or a path under the home directory.
"""
from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
import time
from collections.abc import Callable, Mapping, Sequence
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def out_dir(name: str) -> Path:
    """A directory for a tool's output, under the system's temporary directory."""
    path = Path(tempfile.gettempdir()) / "go-llmprovider-sdk-tools" / name
    path.mkdir(parents=True, exist_ok=True)
    return path


def run(args: Sequence[str], cwd: Path = ROOT, env: Mapping[str, str] | None = None,
        check: bool = False) -> subprocess.CompletedProcess[str]:
    """Run args, never through a shell, with its output captured as text."""
    full_env = None if env is None else {**os.environ, **env}
    return subprocess.run(list(args), cwd=cwd, env=full_env, capture_output=True, text=True, check=check)


def load_limit() -> float:
    """The load average below which a heavy check may start: the CPU count."""
    return float(os.cpu_count() or 1)


def wait_load(limit: float, timeout: float, *, load: Callable[[], float] = lambda: os.getloadavg()[0],
              sleep: Callable[[float], None] = time.sleep, clock: Callable[[], float] = time.monotonic,
              interval: float = 15.0) -> bool:
    """Wait until the one-minute load average is below limit.

    It reports False if timeout seconds pass first. The load-sensitive
    tests in 0034-PLAN Phase 4e, amended, are why heavy checks wait.
    """
    deadline = clock() + timeout
    while load() >= limit:
        if clock() >= deadline:
            return False
        sleep(interval)
    return True


def scratch_clone(src: Path, dest: Path) -> list[str]:
    """Copy src's working tree into a fresh shared clone at dest, staged and committed there.

    It returns the paths the commit changes, deletions excluded. Untracked files
    that are not ignored are included, and a staged or unstaged deletion is
    removed. src's own index is never touched. The commit is made with
    `git commit-tree`, so no hook runs.
    """
    if dest.exists():
        shutil.rmtree(dest)
    run(["git", "clone", "--quiet", "--shared", str(src), str(dest)], cwd=src, check=True)
    listed = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "-z"], cwd=src,
                            capture_output=True, check=True).stdout.decode()
    for rel in filter(None, listed.split("\0")):
        source = src / rel
        if source.is_file():
            (dest / rel).parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, dest / rel)
        elif not source.exists():
            (dest / rel).unlink(missing_ok=True)
    for rel in run(["git", "diff", "--name-only", "--diff-filter=D", "HEAD"], cwd=src, check=True).stdout.split():
        (dest / rel).unlink(missing_ok=True)
    run(["git", "add", "-A"], cwd=dest, check=True)
    changed = run(["git", "diff", "--cached", "--name-only", "--diff-filter=d"], cwd=dest, check=True).stdout.split()
    tree = run(["git", "write-tree"], cwd=dest, check=True).stdout.strip()
    ident = {"GIT_AUTHOR_NAME": "tools", "GIT_AUTHOR_EMAIL": "tools@invalid",
             "GIT_COMMITTER_NAME": "tools", "GIT_COMMITTER_EMAIL": "tools@invalid"}
    commit = run(["git", "commit-tree", tree, "-p", "HEAD", "-m", "scratch working tree"], cwd=dest, env=ident,
                 check=True).stdout.strip()
    run(["git", "update-ref", "HEAD", commit], cwd=dest, check=True)
    return sorted(changed)
