# Supplementary SQLite source-tree integration

Run explicitly with a **local official source archive**; the test never downloads:

```sh
scripts/test-sqlite.sh /absolute/path/to/sqlite-src-3500400.zip
# Retain successful builds, logs, and test databases as well:
KEEP_WORK=1 scripts/test-sqlite.sh /absolute/path/to/sqlite-src-3500400.zip
```

Source: <https://sqlite.org/2025/sqlite-src-3500400.zip>, SQLite **3.50.4**,
check-in `4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3`.
Required archive SHA-256:
`b7b4dc060f36053902fb65b344bbbed592e64b2291a26ac06fe77eec097850e9`.
The script checks the archive, version, check-in, repository integrity manifest,
and byte identity of all six pinned source files before compiling anything.

Requirements: Bash, the locally installed Go toolchain from `go.mod`, a native
C toolchain (`CC` may select one compiler executable), make, unzip, standard
Unix/coreutils/diff tools, and Tcl runtime **and development files**, including
`tclConfig.sh`. SQLite configure must detect Tcl support. Missing requirements
are failures, never silent skips. Optional libraries detected by upstream's
default configure may vary by host; the full configuration is retained.

The script sets `LC_ALL=C`, `TZ=UTC`, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`,
`GOPROXY=off`, and `GOSUMDB=off`, clears ambient compiler/feature/make overrides,
and runs upstream `configure` with **no options**, then `make all testfixture`.
It reads the configured makefile's expanded `OPT_FEATURE_FLAGS` and `OPTS` and
uses the upstream recipe `./lemon FLAGS -S parse.y` in equivalent isolated
directories with the pristine template. The C oracle is compiled from pristine
archive `tool/lemon.c`, without instrumentation or extra compiler flags. The Go
executable is built with cgo disabled. Both runs capture stdout, stderr, and
status. Inventories and every generated `parse.*` artifact (excluding the input
`parse.y`) must match strictly **and** under `diff -bw`; all four expected parser
outputs are required. Whitespace-only differences fail too.

Only the extracted build directory's Lemon executable and parser outputs are
replaced with Go's. The original SQLite sources and make recipes remain intact.
The script checks that neither executable nor parser outputs were overwritten
during the default build, which compiles the parser into SQLite's amalgamation,
libraries, CLI, and upstream Tcl testfixture. It records the running CLI's
version, source identifier, and compile options. The testfixture uses upstream's
test-specific compile flags, so it supplements rather than replaces the default
production-library build.

`upstream-tests.txt` selects unmodified upstream Tcl regressions for syntax
errors and legacy schema parsing, expression precedence, SELECT and compound
queries, CTEs, windows, UPSERT, RETURNING, generated columns, ALTER, triggers,
conflict handling, and transactions. Each file runs in a separate directory;
its exit status and a nonempty zero-error test summary are required. Upstream
capability-based omissions are preserved in its logs. This is not the complete
SQLite `test`, `devtest`, or `releasetest`, a feature matrix, a sanitizer run,
or evidence of cross-platform compatibility. Ordinary `go test` does not run it.
Successful compilation/runtime checks do not substitute for artifact acceptance.

Scratch directories use `${TMPDIR:-/tmp}/lemon-sqlite.XXXXXXXX`, never the review
artifact directory. Failures retain the complete source/build trees, binaries,
parser artifacts, byte and whitespace diffs, configure/make/build/test stdout,
stderr, statuses, test databases, expanded generation command and flags,
`environment.txt`, and a shell command trace (`commands.log`). The error prints
the retained directory and shell-quoted full-script reproduction command.
Inspect `configure.stdout`, `make.stdout`, and `tests/NAME/stdout.txt` for the
earliest failing stage; rerun the printed command to create a fresh isolated run.
The local archive must remain available at its original path for reproduction.
