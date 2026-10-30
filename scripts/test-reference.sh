#!/usr/bin/env bash
# Two independent builds/runs must match the goldens and each other.
set -euo pipefail
export LC_ALL=C TZ=UTC
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
for tool in mktemp sha256sum; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required reference test tool not found: $tool" >&2
    exit 2
  fi
done
work="$(mktemp -d "${TMPDIR:-/tmp}/lemon-reference.XXXXXXXX")"
cleanup() {
  status=$?
  if ((status == 0)); then
    rm -rf -- "$work"
  else
    echo "Reference test failed; full artifacts retained at $work" >&2
    echo "Reproduce: $root/scripts/test-reference.sh" >&2
  fi
}
trap cleanup EXIT
(cd "$root/testdata/golden/sqlite" && sha256sum --check SHA256SUMS)
for run in first second; do
  "$root/scripts/generate-reference.sh" "$work/$run"
  "$root/scripts/compare-artifacts.sh" "$root/testdata/golden/sqlite" "$work/$run"
done
"$root/scripts/compare-artifacts.sh" "$work/first" "$work/second"
echo 'PASS: two independent C builds/runs reproduce every golden artifact byte for byte'
