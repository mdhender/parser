#!/usr/bin/env bash
# Full artifacts from pristine C; compact C-derived hashes for offline Go tests.
set -euo pipefail
export LC_ALL=C TZ=UTC GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=0
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
mode="${1:-differential}"
if [[ $# -gt 1 || ! "$mode" =~ ^(differential|--reference-only|--update-goldens)$ ]]; then
  echo "Usage: $0 [differential|--reference-only|--update-goldens]" >&2; exit 2
fi
compiler="${CC:-cc}"
tools=("$compiler" mktemp sha256sum cmp diff find sort cp mkdir rm dirname uname)
if [[ "$mode" == differential ]]; then tools+=(go); fi
for tool in "${tools[@]}"; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required generation test tool not found: $tool" >&2; exit 2
  fi
done
work="$(mktemp -d "${TMPDIR:-/tmp}/lemon-generation.XXXXXXXX")"
cleanup() {
  status=$?
  if ((status == 0)); then rm -rf -- "$work"; else
    echo "Generation test failed; full artifacts retained at $work" >&2
    printf 'Reproduce: %q %q\n' "$root/scripts/test-generation.sh" "$mode" >&2
  fi
}
trap cleanup EXIT
{
  printf 'LC_ALL=%s\nTZ=%s\nCGO_ENABLED=%s\nCC=%s\n' "$LC_ALL" "$TZ" "$CGO_ENABLED" "$compiler"
  uname -sm
  "$compiler" --version
  if [[ "$mode" == differential ]]; then go version; fi
  printf 'Build: %q -o c-lemon %q\n' "$compiler" "$root/reference/sqlite/tool/lemon.c"
  printf 'Capture cwd: CASE/{crun,gorun,repeat}; argv: ./lemon [options] grammar.y (baseline: parse.y)\n'
} >"$work/environment.txt"
cp "$root/testdata/generation/cases.tsv" "$work/cases.tsv"
(cd "$root/reference" && sha256sum --check SHA256SUMS)
"$compiler" -o "$work/c-lemon" "$root/reference/sqlite/tool/lemon.c"
if [[ "$mode" == differential ]]; then
  (cd "$root" && go build -o "$work/go-lemon" ./cmd/lemon)
fi
output_files() {
  (cd "$1" && find . -type f ! -path ./lemon ! -path ./grammar.y ! -path ./parse.y \
    ! -path ./lempar.c ! -path ./grammar.lt ! -path ./bin/lemon ! -path ./bin/lempar.c | sort)
}
hashes() {
  local dir="$1" file
  while IFS= read -r file; do (cd "$dir" && sha256sum "$file"); done < <(output_files "$dir")
}
compare() {
  local expected="$1" actual="$2" file failed=0
  output_files "$expected" >"$work/expected-files.txt"
  output_files "$actual" >"$work/actual-files.txt"
  diff -u "$work/expected-files.txt" "$work/actual-files.txt" || return 1
  while IFS= read -r file; do
    if ! cmp -s "$expected/$file" "$actual/$file"; then
      echo "Byte mismatch: $file ($expected vs $actual)" >&2
      diff -u "$expected/$file" "$actual/$file" || true
      failed=1
    fi
    if ! diff -bw "$expected/$file" "$actual/$file"; then failed=1; fi
  done <"$work/expected-files.txt"
  return "$failed"
}
capture() {
  local dir="$1" executable="$2" source="$3" operation="$4" filename=grammar.y command=./lemon
  shift 4
  mkdir -p -- "$dir/out"
  cp -- "$executable" "$dir/lemon"
  if [[ "$operation" == baseline ]]; then filename=parse.y; fi
  if [[ "$source" != missing ]]; then cp -- "$source" "$dir/$filename"; fi
  case "$operation" in
    generate|baseline) cp "$root/reference/sqlite/tool/lempar.c" "$dir/lempar.c" ;;
    adjacent) cp "$root/reference/sqlite/tool/lempar.c" "$dir/grammar.lt" ;;
    executable)
      mkdir "$dir/bin"
      cp "$executable" "$dir/bin/lemon"
      cp "$root/reference/sqlite/tool/lempar.c" "$dir/bin/lempar.c"
      command=./bin/lemon ;;
    *) echo "Unknown generation operation: $operation" >&2; exit 2 ;;
  esac
  local status=0
  (cd "$dir" && "$command" "$@" "$filename") >"$dir/stdout.txt" 2>"$dir/stderr.txt" || status=$?
  printf '%s\n' "$status" >"$dir/exit-status.txt"
  if ((status >= 128)); then echo "Abnormal termination ($status): $dir" >&2; return 1; fi
}
count=0
while IFS=$'\t' read -r name operation input options; do
  [[ -z "$name" || "$name" == \#* ]] && continue
  source="$root/testdata/generation/$input"
  if [[ "$input" == sqlite ]]; then source="$root/reference/sqlite/src/parse.y"; fi
  if [[ "$input" == missing ]]; then source=missing; fi
  args=()
  if [[ -n "$options" ]]; then read -r -a args <<<"$options"; fi
  capture "$work/$name/crun" "$work/c-lemon" "$source" "$operation" "${args[@]}"
  hashes "$work/$name/crun" >"$work/$name/SHA256SUMS"
  expected="$root/testdata/generation/golden/$name.sha256"
  if [[ "$mode" == --update-goldens ]]; then
    mkdir -p "$root/testdata/generation/golden"
    cp "$work/$name/SHA256SUMS" "$expected"
  fi
  diff -u "$expected" "$work/$name/SHA256SUMS"
  if [[ "$operation" == baseline ]]; then
    for file in parse.c parse.h parse.out parse.sql stdout.txt stderr.txt exit-status.txt; do
      cmp "$root/testdata/golden/sqlite/$file" "$work/$name/crun/$file"
      diff -bw "$root/testdata/golden/sqlite/$file" "$work/$name/crun/$file"
    done
  fi
  if [[ "$mode" == differential ]]; then
    capture "$work/$name/gorun" "$work/go-lemon" "$source" "$operation" "${args[@]}"
    compare "$work/$name/crun" "$work/$name/gorun"
  else
    capture "$work/$name/repeat" "$work/c-lemon" "$source" "$operation" "${args[@]}"
    compare "$work/$name/crun" "$work/$name/repeat"
  fi
  if [[ "$name" == runtime ]]; then
    runs=(crun repeat)
    if [[ "$mode" == differential ]]; then runs=(crun gorun); fi
    for run in "${runs[@]}"; do
      "$compiler" -o "$work/$name/$run-parser" "$work/$name/$run/grammar.c"
      "$work/$name/$run-parser" >"$work/$name/$run-runtime.txt"
      printf 'value=1 accepted=1 errors=0\n' >"$work/$name/expected-runtime.txt"
      cmp "$work/$name/expected-runtime.txt" "$work/$name/$run-runtime.txt"
    done
    echo 'PASS: generated C runtime (7 - 2 * 3 = 1, accepted once, no errors)'
  fi
  echo "PASS: $name (byte-for-byte and diff -bw)"
  count=$((count + 1))
done <"$root/testdata/generation/cases.tsv"
echo "PASS: $count generation cases ($mode)"
