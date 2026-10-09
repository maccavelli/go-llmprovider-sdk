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
# Step 2 runs golangci-lint with this repository's .golangci.yml instead of
# golint, as ocp-login's script does ("Amendment 2026-09-29 (fourth)"). golint
# is archived, and CI already runs golangci-lint; a gate weaker than CI is not a
# gate. golint's own checks live on as revive's exported, package-comments and
# var-naming rules in .golangci.yml.
#
# Usage:
#   scripts/go-precheck.sh [file.go ...]
#
# With no arguments it checks every tracked Go file; with arguments, only those
# (non-Go arguments are ignored, so callers can pass a whole changed-file list).
# The lint step is package-scoped either way: golangci-lint analyses packages,
# not files, so narrowing it to a file list would report different findings than
# `make lint` and the two would drift.
#
# Exit codes: 0 all clear · 1 a check failed · 2 a required tool is missing.
#
# Env:
#   GOLANGCI_LINT=<path>      golangci-lint binary (default: $(go env GOPATH)/bin)
#   GO_PRECHECK_SKIP_VULN=1   skip govulncheck (offline work)
set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT" || exit 1

# Collect the Go files to check. A named .go path that no longer exists is a
# deletion: it has nothing to format, but its package must still build, so
# its directory is checked (0026-MADR F15).
files=()
deleted_dirs=()
if [ "$#" -gt 0 ]; then
  for f in "$@"; do
    case "$f" in
    *.go)
      if [ -f "$f" ]; then
        files+=("$f")
      else
        deleted_dirs+=("$(dirname "$f")")
      fi
      ;;
    esac
  done
else
  while IFS= read -r f; do
    [ -n "$f" ] && files+=("$f")
  done < <(git ls-files '*.go')
fi

# Nothing to check is not an error: a docs-only change, or a tree with no Go
# code yet. Return before any tool runs — gofmt with no file reads stdin.
if [ "${#files[@]}" -eq 0 ] && [ "${#deleted_dirs[@]}" -eq 0 ]; then
  echo "go-precheck: no Go files to check."
  exit 0
fi

need() {
  command -v "$1" >/dev/null 2>&1 && return 0
  echo "go-precheck: $1 not found in PATH." >&2
  return 1
}

failed=0
fail() {
  # 2 (tool missing) outranks 1 (check failed).
  [ "$failed" -lt "$1" ] && failed="$1"
}

# 1. gofmt, over the files that exist.
need gofmt || exit 2
unformatted=""
if [ "${#files[@]}" -gt 0 ]; then
  unformatted="$(gofmt -l "${files[@]}")"
fi
if [ -n "$unformatted" ]; then
  echo "gofmt: these files are not formatted (run 'gofmt -w <file>'):" >&2
  printf '%s\n' "$unformatted" | sed 's/^/  /' >&2
  fail 1
fi

# 2. golangci-lint, with this repository's configuration: the same commands
# `make lint` runs, so a commit cannot pass a weaker check than CI applies. Like
# `make lint`, it runs for the host and for Windows, whose _windows.go files a
# host-only run never sees (0010-PLAN P6a).
need go || exit 2
GOLANGCI="${GOLANGCI_LINT:-$(go env GOPATH)/bin/golangci-lint}"
if [ -x "$GOLANGCI" ]; then
  for lint_goos in "$(go env GOOS)" windows; do
    if ! lint_out="$(GOOS="$lint_goos" "$GOLANGCI" run -c .golangci.yml --build-tags live_gateways ./... 2>&1)"; then
      echo "golangci-lint (GOOS=$lint_goos):" >&2
      printf '%s\n' "$lint_out" | tail -40 | sed 's/^/  /' >&2
      fail 1
    fi
  done
else
  echo "go-precheck: golangci-lint not found at $GOLANGCI." >&2
  echo "  install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0" >&2
  fail 2
fi

# 3. go vet and go test, over the packages the files belong to, and those a
# deleted file left (./... with no arguments). AGENTS.md requires both on the
# touched packages. A directory a deletion left with no Go file has no package
# to name, so its importers are found by checking the whole module.
if [ "$#" -gt 0 ]; then
  pkgs=()
  whole=0
  while IFS= read -r d; do
    [ -n "$d" ] || continue
    if compgen -G "$d/*.go" >/dev/null; then
      pkgs+=("./$d")
    else
      whole=1
    fi
  done < <({ for f in "${files[@]+"${files[@]}"}"; do dirname "$f"; done; printf '%s\n' "${deleted_dirs[@]+"${deleted_dirs[@]}"}"; } | sort -u)
  if [ "$whole" -eq 1 ]; then
    pkgs=("./...")
  fi
else
  pkgs=("./...")
fi
if ! vet_out="$(go vet "${pkgs[@]}" 2>&1)"; then
  echo "go vet:" >&2
  printf '%s\n' "$vet_out" | sed 's/^/  /' >&2
  fail 1
fi
if ! test_out="$(go test -race "${pkgs[@]}" 2>&1)"; then
  echo "go test -race:" >&2
  printf '%s\n' "$test_out" | tail -40 | sed 's/^/  /' >&2
  fail 1
fi
# go.mod and go.sum change only with the code that needs them (AGENTS.md
# "Dependencies"; 0021-MADR Z5).
if ! tidy_out="$(go mod tidy -diff 2>&1)"; then
  echo "go mod tidy -diff:" >&2
  printf '%s\n' "$tidy_out" | tail -40 | sed 's/^/  /' >&2
  fail 1
fi

# 4. govulncheck, over the module. It reports *called* vulnerabilities, so it is
# a property of the whole build rather than of the edited files. It runs at
# CI's pinned version, read from the workflow, through `go run`, as `make vuln`
# does, so no installed binary of another version decides (0026-MADR F60).
#
# Exit status 3 is govulncheck's "vulnerabilities found" and always fails. Only
# another non-zero status whose output looks like a network failure is treated
# as an unreachable database: matching those words on a status-3 run would let a
# finding whose trace names a `proxy` package or a `Timeout` function through.
vuln_pin="$(grep -oE 'golang.org/x/vuln/cmd/govulncheck@v[0-9][0-9.]*' .github/workflows/ci.yml 2>/dev/null | head -1)"
if [ "${GO_PRECHECK_SKIP_VULN:-0}" = "1" ]; then
  echo "govulncheck: skipped (GO_PRECHECK_SKIP_VULN=1)" >&2
elif [ -z "$vuln_pin" ]; then
  echo "govulncheck: no pin in .github/workflows/ci.yml" >&2
  fail 2
else
  vuln_out="$(go run "$vuln_pin" ./... 2>&1)"
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
fi

if [ "$failed" -eq 0 ]; then
  echo "go-precheck: ${#files[@]} file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)."
fi
exit "$failed"
