#!/usr/bin/env bash
# Compare the complete output contract of the pinned SQLite invocation.
set -euo pipefail
export LC_ALL=C
if [[ $# != 2 ]]; then
  echo "Usage: $0 EXPECTED_DIRECTORY ACTUAL_DIRECTORY" >&2
  exit 2
fi
for tool in cmp diff; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required comparison tool not found: $tool" >&2
    exit 2
  fi
done
for artifact in parse.c parse.h parse.out parse.sql stdout.txt stderr.txt exit-status.txt; do
  expected="$1/$artifact"
  actual="$2/$artifact"
  if [[ ! -f "$expected" || ! -f "$actual" ]]; then
    echo "Missing required artifact: $artifact in $1 or $2" >&2
    exit 1
  fi
  if cmp -s "$expected" "$actual"; then
    # Exercise both acceptance comparisons even when bytes match.
    diff -bw "$expected" "$actual"
    echo "$artifact: byte-for-byte and diff -bw match"
  else
    echo "$artifact: byte-level mismatch (strict baseline comparison fails)" >&2
    diff -u "$expected" "$actual" || true
    if diff -bw "$expected" "$actual"; then
      echo 'diff -bw matches, but byte differences still require review.' >&2
    fi
    exit 1
  fi
done
