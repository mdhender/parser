#!/usr/bin/env bash
# Build and run the pristine, pinned C oracle. Never downloads or updates goldens.
set -euo pipefail
export LC_ALL=C TZ=UTC
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ $# != 1 ]]; then
  echo "Usage: $0 NEW_OUTPUT_DIRECTORY" >&2
  exit 2
fi
compiler="${CC:-cc}"
for tool in "$compiler" sha256sum cp mkdir; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required reference tool not found: $tool" >&2
    exit 2
  fi
done
(cd "$root/reference" && sha256sum --check SHA256SUMS)
# Refuse existing directories so stale artifacts cannot satisfy a comparison.
mkdir -- "$1"
output="$(cd -- "$1" && pwd)"
{
  printf 'LC_ALL=%s\nTZ=%s\nCC=%s\n' "$LC_ALL" "$TZ" "$compiler"
  printf 'OPT_FEATURE_FLAGS=\nOPTS=\n'
  uname -sm
  "$compiler" --version
  printf 'Build: %q -o lemon %q\n' "$compiler" "$root/reference/sqlite/tool/lemon.c"
  printf 'Generate: ./lemon -S parse.y\n'
} >"$output/environment.txt"
# Match main.mk's host-compiler rule; intentionally no additional C flags.
if ! "$compiler" -o "$output/lemon" "$root/reference/sqlite/tool/lemon.c" \
    >"$output/build.stdout.txt" 2>"$output/build.stderr.txt"; then
  cat "$output/build.stderr.txt" >&2
  echo "Reference build failed; artifacts retained at $output" >&2
  exit 1
fi
cp "$root/reference/sqlite/tool/lempar.c" "$output/lempar.c"
cp "$root/reference/sqlite/src/parse.y" "$output/parse.y"
status=0
(cd "$output" && ./lemon -S parse.y) \
  >"$output/stdout.txt" 2>"$output/stderr.txt" || status=$?
printf '%s\n' "$status" >"$output/exit-status.txt"
if ((status != 0)); then
  cat "$output/stderr.txt" >&2
  echo "Reference generation failed ($status); artifacts retained at $output" >&2
  exit 1
fi
echo "Reference artifacts generated at $output"
