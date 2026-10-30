#!/usr/bin/env bash
# Offline Milestone C oracle: stops before compression and state resorting.
# --update-goldens regenerates from C and independently repeats every case.
set -euo pipefail
export LC_ALL=C TZ=UTC GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=0
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
mode="${1:-differential}"
if [[ $# -gt 1 || ! "$mode" =~ ^(differential|--reference-only|--update-goldens)$ ]]; then
  echo "Usage: $0 [differential|--reference-only|--update-goldens]" >&2
  exit 2
fi
compiler="${CC:-cc}"
tools=("$compiler" mktemp sha256sum cmp diff find sort cp mkdir rm dirname uname)
if [[ "$mode" == differential ]]; then tools+=(go); fi
for tool in "${tools[@]}"; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required automaton test tool not found: $tool" >&2
    exit 2
  fi
done
work="$(mktemp -d "${TMPDIR:-/tmp}/lemon-automaton.XXXXXXXX")"
cleanup() {
  status=$?
  if ((status == 0)); then
    rm -rf -- "$work"
  else
    echo "Automaton test failed; full artifacts retained at $work" >&2
    printf 'Reproduce: %q %q\n' "$root/scripts/test-automaton.sh" "$mode" >&2
  fi
}
trap cleanup EXIT
{
  printf 'LC_ALL=%s\nTZ=%s\nCGO_ENABLED=%s\nCC=%s\n' "$LC_ALL" "$TZ" "$CGO_ENABLED" "$compiler"
  uname -sm
  "$compiler" --version
  if [[ "$mode" == differential ]]; then go version; fi
  printf 'Snapshot build: %q -o snapshot %q\n' "$compiler" "$root/reference/automaton-snapshot.c"
  printf 'Cases: %q\n' "$root/testdata/automaton/cases.tsv"
  printf 'Capture cwd: CASE/{crun,gorun,repeat}; argv: ./lemon [options] grammar.y\n'
} >"$work/environment.txt"
cp "$root/testdata/automaton/cases.tsv" "$work/cases.tsv"
(cd "$root/reference" && sha256sum --check SHA256SUMS)
"$compiler" -o "$work/snapshot" "$root/reference/automaton-snapshot.c"
if [[ "$mode" == differential ]]; then
  (cd "$root" && go build -o "$work/go-lemon" ./cmd/lemon)
fi

# Inventory everything except the executable and input, including stdout,
# stderr, exit status and any unexpected artifacts. No normalization.
output_files() {
  (cd "$1" && find . -type f ! -path ./lemon ! -path ./grammar.y | sort)
}
compare() {
  local expected="$1" actual="$2" file failed=0
  output_files "$expected" >"$work/expected-files.txt"
  output_files "$actual" >"$work/actual-files.txt"
  if ! diff -u "$work/expected-files.txt" "$work/actual-files.txt"; then
    echo "Output file inventory differs: $expected vs $actual" >&2
    return 1
  fi
  while IFS= read -r file; do
    if ! cmp -s "$expected/$file" "$actual/$file"; then
      echo "Byte mismatch: $file ($expected vs $actual)" >&2
      diff -u "$expected/$file" "$actual/$file" || true
      failed=1
    fi
    if ! diff -bw "$expected/$file" "$actual/$file"; then
      echo "diff -bw mismatch: $file" >&2
      failed=1
    fi
  done <"$work/expected-files.txt"
  return "$failed"
}
capture() {
  local dir="$1" executable="$2" source="$3"
  shift 3
  mkdir -p -- "$dir"
  cp -- "$executable" "$dir/lemon"
  if [[ "$source" != missing ]]; then cp -- "$source" "$dir/grammar.y"; fi
  local status=0
  (cd "$dir" && ./lemon "$@" grammar.y) >"$dir/stdout.txt" 2>"$dir/stderr.txt" || status=$?
  printf '%s\n' "$status" >"$dir/exit-status.txt"
  # Crashes are oracle failures, never acceptable golden behavior.
  if ((status >= 128)); then
    echo "Oracle or implementation terminated abnormally (status $status): $dir" >&2
    return 1
  fi
}
count=0
while IFS=$'\t' read -r name operation input options; do
  [[ -z "$name" || "$name" == \#* ]] && continue
  if [[ "$operation" != automaton ]]; then
    echo "Unknown manifest operation: $operation" >&2
    exit 2
  fi
  source="$root/testdata/automaton/fixtures/$input"
  if [[ "$input" == sqlite ]]; then source="$root/reference/sqlite/src/parse.y"; fi
  if [[ "$input" == missing ]]; then source=missing; fi
  args=()
  # Manifest options are whitespace-free argv words, not shell code.
  if [[ -n "$options" ]]; then read -r -a args <<<"$options"; fi
  expected="$root/testdata/automaton/golden/$name"
  capture "$work/$name/crun" "$work/snapshot" "$source" "${args[@]}"
  if [[ "$mode" == --update-goldens ]]; then
    mkdir -p -- "$expected"
    # Do not delete stale goldens: inventory comparison must expose them.
    output_files "$work/$name/crun" >"$work/copy-files.txt"
    while IFS= read -r file; do
      mkdir -p -- "$expected/$(dirname -- "$file")"
      cp -- "$work/$name/crun/$file" "$expected/$file"
    done <"$work/copy-files.txt"
  fi
  compare "$expected" "$work/$name/crun"
  if [[ "$mode" == differential ]]; then
    capture "$work/$name/gorun" "$work/go-lemon" "$source" --automaton-snapshot "${args[@]}"
    compare "$work/$name/crun" "$work/$name/gorun"
  else
    capture "$work/$name/repeat" "$work/snapshot" "$source" "${args[@]}"
    compare "$work/$name/crun" "$work/$name/repeat"
  fi
  echo "PASS: $name (byte-for-byte and diff -bw)"
  count=$((count + 1))
done <"$root/testdata/automaton/cases.tsv"
echo "PASS: $count automaton cases ($mode)"
