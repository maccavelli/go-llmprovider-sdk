#!/usr/bin/env bash
# Go pre-add checks: format, lint, vet, tests, vulnerabilities.
#
# The single implementation of the pre-add rule in AGENTS.md, called from two
# places so they cannot drift: `make pre-add-check`, and the machine-wide agent
# gate that runs before every agent `git commit`
# (~/.agents/hooks/lib/precommit-checks.sh), which prefers this script whenever
# the repository ships one and Go files are staged.
#
# Adapted from magic-cli-remote's scripts/go-precheck.sh under
# docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md,
# "Amendment 2026-09-29: pre-add gate and agent pointers".
#
# Usage:
#   scripts/go-precheck.sh [file.go ...]
#
# With no arguments it checks every tracked Go file; with arguments, only those
# (non-Go arguments are ignored, so callers can pass a whole changed-file list).
#
# Exit codes: 0 all clear · 1 a check failed · 2 a required tool is missing.
#
# Env:
#   GO_PRECHECK_SKIP_VULN=1   skip govulncheck (offline work)
set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT" || exit 1

# Collect the Go files to check.
files=()
if [ "$#" -gt 0 ]; then
  for f in "$@"; do
    case "$f" in
    *.go) [ -f "$f" ] && files+=("$f") ;;
    esac
  done
else
  while IFS= read -r f; do
    [ -n "$f" ] && files+=("$f")
  done < <(git ls-files '*.go')
fi

# Nothing to check is not an error: a docs-only change, or a tree with no Go
# code yet. Return before any tool runs — gofmt with no file reads stdin.
if [ "${#files[@]}" -eq 0 ]; then
  echo "go-precheck: no Go files to check."
  exit 0
fi

need() {
  command -v "$1" >/dev/null 2>&1 && return 0
  echo "go-precheck: $1 not found in PATH." >&2
  case "$1" in
  golint) echo "  install: go install golang.org/x/lint/golint@latest" >&2 ;;
  govulncheck) echo "  install: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2 ;;
  esac
  return 1
}

failed=0
fail() {
  # 2 (tool missing) outranks 1 (check failed).
  [ "$failed" -lt "$1" ] && failed="$1"
}

# 1. gofmt.
need gofmt || exit 2
unformatted="$(gofmt -l "${files[@]}")"
if [ -n "$unformatted" ]; then
  echo "gofmt: these files are not formatted (run 'gofmt -w <file>'):" >&2
  echo "$unformatted" | sed 's/^/  /' >&2
  fail 1
fi

# 2. golint, per file so the output names what to fix.
if need golint; then
  lint_out=""
  for f in "${files[@]}"; do
    out="$(golint "$f" 2>&1)"
    [ -n "$out" ] && lint_out+="$out"$'\n'
  done
  if [ -n "$lint_out" ]; then
    echo "golint:" >&2
    printf '%s' "$lint_out" | sed 's/^/  /' >&2
    fail 1
  fi
else
  fail 2
fi

# 3. go vet and go test, over the packages the files belong to (./... with no
# arguments). AGENTS.md requires both on the touched packages.
need go || exit 2
if [ "$#" -gt 0 ]; then
  pkgs=()
  while IFS= read -r d; do
    [ -n "$d" ] && pkgs+=("./$d")
  done < <(for f in "${files[@]}"; do dirname "$f"; done | sort -u)
else
  pkgs=("./...")
fi
if ! vet_out="$(go vet "${pkgs[@]}" 2>&1)"; then
  echo "go vet:" >&2
  printf '%s\n' "$vet_out" | sed 's/^/  /' >&2
  fail 1
fi
if ! test_out="$(go test "${pkgs[@]}" 2>&1)"; then
  echo "go test:" >&2
  printf '%s\n' "$test_out" | tail -40 | sed 's/^/  /' >&2
  fail 1
fi

# 4. govulncheck, over the module. It reports *called* vulnerabilities, so it is
# a property of the whole build rather than of the edited files.
#
# Exit status 3 is govulncheck's "vulnerabilities found" and always fails. Only
# another non-zero status whose output looks like a network failure is treated
# as an unreachable database: matching those words on a status-3 run would let a
# finding whose trace names a `proxy` package or a `Timeout` function through.
if [ "${GO_PRECHECK_SKIP_VULN:-0}" = "1" ]; then
  echo "govulncheck: skipped (GO_PRECHECK_SKIP_VULN=1)" >&2
elif need govulncheck; then
  vuln_out="$(govulncheck ./... 2>&1)"
  vuln_rc=$?
  if [ "$vuln_rc" -eq 3 ]; then
    echo "govulncheck: vulnerabilities found:" >&2
    printf '%s\n' "$vuln_out" | tail -30 | sed 's/^/  /' >&2
    fail 1
  elif [ "$vuln_rc" -ne 0 ]; then
    if printf '%s' "$vuln_out" | grep -qiE 'no such host|connection refused|timeout|dial tcp|proxy'; then
      echo "govulncheck: could not reach the vulnerability database; skipped." >&2
    else
      echo "govulncheck: failed (exit $vuln_rc):" >&2
      printf '%s\n' "$vuln_out" | tail -30 | sed 's/^/  /' >&2
      fail 1
    fi
  fi
else
  fail 2
fi

if [ "$failed" -eq 0 ]; then
  echo "go-precheck: ${#files[@]} file(s) clean (gofmt, golint, go vet, go test, govulncheck)."
fi
exit "$failed"
