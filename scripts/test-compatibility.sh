#!/usr/bin/env bash
# Final acceptance against the immutable SQLite pin, without downloading inputs.
set -euo pipefail
export LC_ALL=C TZ=UTC GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=0
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
if (( $# > 1 )); then
  echo "Usage: $0 [SQLITE_SOURCE_ARCHIVE]" >&2; exit 2
fi
for tool in "${CC:-cc}" go mktemp cp rm mkdir dirname tee readelf grep; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required compatibility test tool not found: $tool" >&2; exit 2
  fi
done
work="$(mktemp -d "${TMPDIR:-/tmp}/lemon-compatibility.XXXXXXXX")"
cleanup() {
  status=$?
  if ((status == 0)); then rm -rf -- "$work"; else
    echo "Compatibility test failed; logs and artifacts retained at $work" >&2
    printf 'Reproduce: %q' "$root/scripts/test-compatibility.sh" >&2
    if (( $# )); then printf ' %q' "$@" >&2; fi
    printf '\n' >&2
  fi
}
trap 'cleanup "$@"' EXIT
(
  cd "$root"
  go version
  CC=/nonexistent/compiler go test -count=3 ./...
  CC=/nonexistent/compiler go vet ./...
  CC=/nonexistent/compiler go build -o "$work/lemon" ./cmd/lemon
) 2>&1 | tee "$work/go.log"

# No ELF interpreter: the Go command does not load a C runtime. Also execute
# normal generation with no external template, compiler, or utilities on PATH.
readelf -l "$work/lemon" >"$work/program-headers.txt"
if grep -q INTERP "$work/program-headers.txt"; then
  echo 'Go Lemon unexpectedly requires a dynamic interpreter' >&2; exit 1
fi
for run in first second; do
  mkdir "$work/$run"
  cp "$work/lemon" "$work/$run/lemon"
  cp "$root/reference/sqlite/src/parse.y" "$work/$run/parse.y"
  status=0
  (cd "$work/$run" && PATH=/nonexistent CC=/nonexistent/compiler ./lemon -S parse.y) \
    >"$work/$run/stdout.txt" 2>"$work/$run/stderr.txt" || status=$?
  printf '%s\n' "$status" >"$work/$run/exit-status.txt"
  "$root/scripts/compare-artifacts.sh" "$root/testdata/golden/sqlite" "$work/$run" \
    2>&1 | tee "$work/$run-comparison.log"
done
"$root/scripts/compare-artifacts.sh" "$work/first" "$work/second"
echo 'PASS: embedded-template Go generation twice, exact SQLite goldens, no dynamic interpreter'

for stage in reference grammar automaton tables generation; do
  "$root/scripts/test-$stage.sh" 2>&1 | tee "$work/$stage.log"
done
if (( $# == 1 )); then
  "$root/scripts/test-sqlite.sh" "$1" 2>&1 | tee "$work/sqlite.log"
else
  echo 'SQLite source-tree integration not requested; run scripts/test-sqlite.sh with the pinned archive separately.'
fi
echo 'PASS: Milestone F pinned-output compatibility (strict bytes and diff -bw)'
