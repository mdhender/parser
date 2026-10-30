#!/usr/bin/env bash
# Supplementary, offline SQLite source-tree integration. Never downloads inputs.
set -euo pipefail
export LC_ALL=C TZ=UTC CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ $# != 1 || ! -f "$1" || ! "${KEEP_WORK:-0}" =~ ^[01]$ ]]; then
  echo "Usage: [KEEP_WORK=1] $0 /path/to/sqlite-src-3500400.zip" >&2
  echo 'Requires the local official archive; no downloads are performed.' >&2
  exit 2
fi
archive="$(cd -- "$(dirname -- "$1")" && pwd)/$(basename -- "$1")"
compiler="${CC:-cc}"
for tool in "$compiler" go make unzip sha256sum cmp diff mktemp cp rm mkdir find sort grep sed ar ranlib tclsh uname; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required SQLite integration tool not found: $tool" >&2
    exit 2
  fi
done
# This is a native, default configure build, not an ambient feature matrix.
unset CFLAGS CPPFLAGS CXXFLAGS LDFLAGS LIBS OPTIONS OPTS OPT_FEATURE_FLAGS
unset MAKEFLAGS MFLAGS MAKEFILES CONFIG_SITE
export CC="$compiler"
work="$(mktemp -d "${TMPDIR:-/tmp}/lemon-sqlite.XXXXXXXX")"
cleanup() {
  status=$?
  if ((status == 0)) && [[ "${KEEP_WORK:-0}" == 0 ]]; then
    rm -rf -- "$work"
  else
    echo "SQLite integration artifacts retained at $work" >&2
    if ((status != 0)); then
      echo 'FAIL: see commands.log, environment.txt, and the stage logs in that directory.' >&2
    fi
    printf 'Reproduce: KEEP_WORK=1 CC=%q %q %q\n' "$compiler" "$root/scripts/test-sqlite.sh" "$archive" >&2
  fi
}
trap cleanup EXIT
exec 9>"$work/commands.log"
export BASH_XTRACEFD=9
set -x
{
  printf 'Archive: %s\nLC_ALL=%s\nTZ=%s\nCGO_ENABLED=%s\nCC=%s\n' "$archive" "$LC_ALL" "$TZ" "$CGO_ENABLED" "$compiler"
  printf 'GOTOOLCHAIN=%s GOPROXY=%s GOSUMDB=%s\n' "$GOTOOLCHAIN" "$GOPROXY" "$GOSUMDB"
  printf 'Configure: no options; CFLAGS/CPPFLAGS/LDFLAGS/LIBS/OPTIONS/OPTS and make overrides unset\n'
  uname -srm
  "$compiler" --version
  go version
  make --version
  tclsh <<<'puts "Tcl [info patchlevel]"'
} >"$work/environment.txt" 2>&1
printf '%s  %s\n' b7b4dc060f36053902fb65b344bbbed592e64b2291a26ac06fe77eec097850e9 "$archive" >"$work/archive.sha256"
sha256sum --check "$work/archive.sha256"
unzip -q "$archive" -d "$work/source"
source="$work/source/sqlite-src-3500400"
[[ "$(cat "$source/manifest.uuid")" == 4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3 ]]
[[ "$(cat "$source/VERSION")" == 3.50.4 ]]
(cd "$root/reference" && sha256sum --check SHA256SUMS)
for file in tool/lemon.c tool/lempar.c src/parse.y main.mk manifest.uuid VERSION; do
  cmp "$root/reference/sqlite/$file" "$source/$file"
done
echo 'PASS: official archive hash, check-in, version, and all pinned source inputs'
cp "$root/testdata/sqlite/upstream-tests.txt" "$work/upstream-tests.txt"
(cd "$root" && go build -o "$work/go-lemon" ./cmd/lemon) >"$work/go-build.stdout" 2>"$work/go-build.stderr"
"$compiler" -o "$work/c-lemon" "$source/tool/lemon.c" >"$work/c-build.stdout" 2>"$work/c-build.stderr"
mkdir "$work/build"
(cd "$work/build" && "$source/configure") >"$work/configure.stdout" 2>"$work/configure.stderr" || {
  echo "SQLite configure failed; inspect $work/configure.{stdout,stderr} (Tcl development headers/libraries are required)." >&2
  exit 1
}
# Expand the real recipe variables rather than hardcoding the host's defaults.
cat >"$work/flags.mk" <<'MAKE'
.PHONY: integration-flags
integration-flags:
	@printf '%s\n' '$(OPT_FEATURE_FLAGS)' '$(OPTS)' '$(HAVE_TCL)' '$(TCL_CONFIG_SH)'
MAKE
(cd "$work/build" && make -s -f Makefile -f "$work/flags.mk" integration-flags) >"$work/parser-flags.txt" 2>"$work/parser-flags.stderr"
mapfile -t settings <"$work/parser-flags.txt"
if [[ "${settings[2]:-}" != 1 || ! -f "${settings[3]:-}" ]]; then
  echo 'Required SQLite Tcl test support not configured: install Tcl 8.6+ runtime and development headers/libraries (including tclConfig.sh).' >&2
  exit 2
fi
args=()
read -r -a feature_args <<<"${settings[0]}"
read -r -a option_args <<<"${settings[1]}"
args+=("${feature_args[@]}" "${option_args[@]}" -S parse.y)
printf './lemon' >"$work/parser-command.txt"
printf ' %q' "${args[@]}" >>"$work/parser-command.txt"
printf '\n' >>"$work/parser-command.txt"
for run in crun gorun; do
  mkdir "$work/$run"
  cp "$source/src/parse.y" "$work/$run/parse.y"
  cp "$source/tool/lempar.c" "$work/$run/lempar.c"
  executable="$work/c-lemon"
  if [[ "$run" == gorun ]]; then executable="$work/go-lemon"; fi
  cp "$executable" "$work/$run/lemon"
  status=0
  (cd "$work/$run" && ./lemon "${args[@]}") >"$work/$run/stdout.txt" 2>"$work/$run/stderr.txt" || status=$?
  printf '%s\n' "$status" >"$work/$run/exit-status.txt"
  (cd "$work/$run" && find . -maxdepth 1 -type f -name 'parse.*' ! -name parse.y | sort) >"$work/$run/artifacts.txt"
done
failed=0
diff -u "$work/crun/artifacts.txt" "$work/gorun/artifacts.txt" >"$work/inventory.diff" || failed=1
for file in parse.c parse.h parse.out parse.sql; do
  [[ -f "$work/crun/$file" && -f "$work/gorun/$file" ]] || failed=1
done
mapfile -t artifacts <"$work/crun/artifacts.txt"
for file in "${artifacts[@]}" stdout.txt stderr.txt exit-status.txt; do
  name="${file#./}"
  if ! cmp -s "$work/crun/$file" "$work/gorun/$file"; then
    echo "Byte mismatch: $file" >&2
    failed=1
  fi
  diff -u "$work/crun/$file" "$work/gorun/$file" >"$work/$name.diff" || failed=1
  diff -bw "$work/crun/$file" "$work/gorun/$file" >"$work/$name.diff-bw" || failed=1
done
[[ "$(cat "$work/crun/exit-status.txt")" == 0 ]] || failed=1
if ((failed)); then echo "Parser comparison failed; inspect $work/*.diff*" >&2; exit 1; fi
echo 'PASS: configured C/Go parser artifacts, stdout, stderr, status (bytes and diff -bw)'
# Satisfy upstream's generator prerequisites, then replace only the build-tree
# executable and outputs. No source-tree file or upstream make recipe is edited.
(cd "$work/build" && make lemon) >"$work/make-lemon.stdout" 2>"$work/make-lemon.stderr"
cp "$work/go-lemon" "$work/build/lemon"
cp "$source/tool/lempar.c" "$work/build/lempar.c"
cp "$source/src/parse.y" "$work/build/parse.y"
for file in parse.c parse.h parse.out parse.sql; do cp "$work/gorun/$file" "$work/build/$file"; done
(cd "$work/build" && make all testfixture) >"$work/make.stdout" 2>"$work/make.stderr" || {
  echo "SQLite default build/testfixture failed; inspect $work/make.{stdout,stderr}. Tcl headers/libraries are required for testfixture." >&2
  exit 1
}
cmp "$work/go-lemon" "$work/build/lemon"
for file in parse.c parse.h parse.out parse.sql; do cmp "$work/gorun/$file" "$work/build/$file"; done
echo 'PASS: default SQLite source-tree build and Tcl testfixture with Go-generated parser'
(cd "$work/build" && ./sqlite3 :memory: 'SELECT sqlite_version(), sqlite_source_id(); PRAGMA compile_options;') >"$work/sqlite-build.txt" 2>"$work/sqlite-build.stderr"
grep -q '^3\.50\.4|.*4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3' "$work/sqlite-build.txt"
mkdir "$work/tests"
count=0
while IFS= read -r test; do
  [[ -z "$test" || "$test" == \#* ]] && continue
  mkdir "$work/tests/$test"
  status=0
  (cd "$work/tests/$test" && "$work/build/testfixture" "$source/test/$test") >"$work/tests/$test/stdout.txt" 2>"$work/tests/$test/stderr.txt" || status=$?
  printf '%s\n' "$status" >"$work/tests/$test/exit-status.txt"
  if ((status != 0)) || ! grep -Eq '^0 errors out of [1-9][0-9]* tests' "$work/tests/$test/stdout.txt"; then
    echo "SQLite regression failed or ran no tests: $test; inspect $work/tests/$test" >&2
    exit 1
  fi
  grep -E ' errors out of .* tests' "$work/tests/$test/stdout.txt"
  echo "PASS: upstream $test"
  count=$((count + 1))
done <"$work/upstream-tests.txt"
echo "PASS: SQLite 3.50.4 default build and $count upstream SQL/parser regression files (supplementary, not full SQLite releasetest)"
