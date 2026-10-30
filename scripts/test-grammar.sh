#!/usr/bin/env bash
# Offline Milestone B oracle. No parser generation or final-artifact claims.
# Default: build cmd/lemon, check C goldens, then compare C against Go.
# --reference-only: reproduce goldens without Go. --update-goldens: explicit
# regeneration from C, followed by a second independent run for determinism.
set -euo pipefail
export LC_ALL=C TZ=UTC GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=0
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
mode="${1:-differential}"
if [[ $# -gt 1 || ! "$mode" =~ ^(differential|--reference-only|--update-goldens)$ ]]; then
  echo "Usage: $0 [--reference-only|--update-goldens]" >&2
  exit 2
fi
compiler="${CC:-cc}"
tools=("$compiler" mktemp sha256sum cmp diff find sort cp mkdir rm dirname uname)
if [[ "$mode" == differential ]]; then tools+=(go); fi
for tool in "${tools[@]}"; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required grammar test tool not found: $tool" >&2
    exit 2
  fi
done
work="$(mktemp -d "${TMPDIR:-/tmp}/lemon-grammar.XXXXXXXX")"
cleanup() {
  status=$?
  if ((status == 0)); then
    rm -rf -- "$work"
  else
    echo "Grammar test failed; full artifacts retained at $work" >&2
    printf 'Reproduce: %q %q\n' "$root/scripts/test-grammar.sh" "$mode" >&2
  fi
}
trap cleanup EXIT
{
  printf 'LC_ALL=%s\nTZ=%s\nCGO_ENABLED=%s\nCC=%s\n' "$LC_ALL" "$TZ" "$CGO_ENABLED" "$compiler"
  uname -sm
  "$compiler" --version
  if [[ "$mode" == differential ]]; then go version; fi
  printf 'Snapshot build: %q -o snapshot %q\n' "$compiler" "$root/reference/grammar-snapshot.c"
  printf 'Pristine build: %q -o pristine %q\n' "$compiler" "$root/reference/sqlite/tool/lemon.c"
  printf 'Cases: %q\n' "$root/testdata/grammar/cases.tsv"
} >"$work/environment.txt"
cp "$root/testdata/grammar/cases.tsv" "$work/cases.tsv"
(cd "$root/reference" && sha256sum --check SHA256SUMS)
"$compiler" -o "$work/snapshot" "$root/reference/grammar-snapshot.c"
"$compiler" -o "$work/pristine" "$root/reference/sqlite/tool/lemon.c"
if [[ "$mode" == differential ]]; then
  (cd "$root" && go build -o "$work/go-lemon" ./cmd/lemon)
fi

# Files excluded from output are only the copied executable and grammar.
# Every other generated file, plus stdout, stderr and status, is compared.
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
}
count=0
while IFS=$'\t' read -r name operation input options; do
  [[ -z "$name" || "$name" == \#* ]] && continue
  source="$root/testdata/grammar/fixtures/$input"
  if [[ "$input" == sqlite ]]; then source="$root/reference/sqlite/src/parse.y"; fi
  if [[ "$input" == missing ]]; then source=missing; fi
  args=()
  # Manifest options are deliberately restricted to whitespace-free argv words.
  if [[ -n "$options" ]]; then read -r -a args <<<"$options"; fi
  goargs=("${args[@]}")
  case "$operation" in
    snapshot) oracle="$work/snapshot"; goargs=(--grammar-snapshot "${args[@]}");;
    reprint) oracle="$work/pristine"; args=(-g "${args[@]}"); goargs=("${args[@]}");;
    preprocess) oracle="$work/pristine"; args=(-E "${args[@]}"); goargs=("${args[@]}");;
    *) echo "Unknown manifest operation: $operation" >&2; exit 2;;
  esac
  expected="$root/testdata/grammar/golden/$name"
  capture "$work/$name/crun" "$oracle" "$source" "${args[@]}"
  if [[ "$mode" == --update-goldens ]]; then
    mkdir -p -- "$expected"
    # Refuse to silently preserve stale or unexpected golden artifacts.
    output_files "$work/$name/crun" >"$work/copy-files.txt"
    while IFS= read -r file; do
      mkdir -p -- "$expected/$(dirname -- "$file")"
      cp -- "$work/$name/crun/$file" "$expected/$file"
    done <"$work/copy-files.txt"
  fi
  compare "$expected" "$work/$name/crun"
  if [[ "$mode" == differential ]]; then
    capture "$work/$name/gorun" "$work/go-lemon" "$source" "${goargs[@]}"
    compare "$work/$name/crun" "$work/$name/gorun"
  else
    capture "$work/$name/repeat" "$oracle" "$source" "${args[@]}"
    compare "$work/$name/crun" "$work/$name/repeat"
  fi
  echo "PASS: $name (byte-for-byte and diff -bw)"
  count=$((count + 1))
done <"$root/testdata/grammar/cases.tsv"
echo "PASS: $count grammar cases ($mode)"
