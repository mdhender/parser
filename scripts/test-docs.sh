#!/usr/bin/env bash
# Execute the tutorial's actual shell block; C checks stay outside offline Go tests.
set -euo pipefail
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
for tool in go cc awk mktemp cp rm diff; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required documentation test tool not found: $tool" >&2
    exit 2
  fi
done
work="$(mktemp -d "${TMPDIR:-/tmp}/lemon-docs.XXXXXXXX")"
trap 'rm -rf "$work"' EXIT
for example in 'lemon calc.y' 'c main.c'; do
  read -r language filename <<<"$example"
  awk -v marker="\`\`\`$language" \
    '$0 == marker { block=1; next } /^```$/ { block=0 } block' \
    "$root/docs/tutorials/first-parser.md" >"$work/$filename"
  diff -u "$root/docs/tutorials/first-parser/$filename" "$work/$filename"
done
awk '/^```sh$/ { block=1; next } /^```$/ { block=0 } block' \
  "$root/docs/tutorials/first-parser.md" >"$work/tutorial.sh"
test -s "$work/tutorial.sh"
(cd "$root" && bash "$work/tutorial.sh") >"$work/actual.txt"
printf 'Result: 7\n' >"$work/expected.txt"
diff -u "$work/expected.txt" "$work/actual.txt"
echo 'PASS: first-parser tutorial generates, compiles, and prints exact expected output'
