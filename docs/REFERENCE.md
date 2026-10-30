# Pinned SQLite reference

## Provenance

Milestone A uses the official **SQLite 3.50.4** release (2025-07-30), a
deliberate fixed baseline, not a claim that this is the latest release.

- Official repository: <https://sqlite.org/src>
- Immutable Fossil check-in: `4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3`
- Check-in page: <https://sqlite.org/src/info/4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3>
- Official source archive: <https://sqlite.org/2025/sqlite-src-3500400.zip>
- Archive SHA-256: `b7b4dc060f36053902fb65b344bbbed592e64b2291a26ac06fe77eec097850e9`

The vendored files under `reference/sqlite/` were copied unchanged from that
archive. Its `manifest.uuid` supplies the check-in identifier; `VERSION`
contains `3.50.4`. `main.mk` preserves the authoritative build recipe from
the same revision. No Go port was used as an implementation source.
During baseline establishment, the primary inputs, `main.mk`, and `VERSION`
were also checked against the archive's Fossil manifest file hashes (SHA3-256).
The manifest itself hashes to the recorded check-in after removing the
archive's explicit final `# Remove this line...` marker.
The SQLite files retain their original public-domain notices; the repository's
MIT license does not replace those notices.

`reference/SHA256SUMS` is the machine-readable source integrity manifest.
SHA-256 values for the three primary inputs are:

| File | SHA-256 |
| --- | --- |
| `tool/lemon.c` | `b0b32c29262047de987dc3c051b1e3f6f06a36f68403825c0c2e3ffecb4c6e00` |
| `tool/lempar.c` | `dc0a1e45bdf5d5413467ff157135d388f552fa90624d9c0b76bee1c55cec427a` |
| `src/parse.y` | `b45f310ae9ad6f5a50afc5713e3117a47d249ce29bb8b7e9300d0e30e80927b3` |

The manifest also pins `main.mk`, `manifest.uuid`, and `VERSION`.
Updating the reference requires an explicit reviewed change to the sources,
manifests, documentation, and golden artifacts. Tests never update them.

## Build recipe and selected configuration

The relevant unmodified excerpts from `reference/sqlite/main.mk` are
lines 1126–1130 and 1462–1468:

```make
# Rules to build the LEMON compiler generator
#
lemon$(B.exe): $(MAKE_SANITY_CHECK) $(TOP)/tool/lemon.c $(TOP)/tool/lempar.c
	$(B.cc) -o $@ $(TOP)/tool/lemon.c
	cp $(TOP)/tool/lempar.c .
```

```make
# Rules to build parse.c and parse.h - the outputs of lemon.
#
parse.h:	parse.c

parse.c:	$(TOP)/src/parse.y lemon$(B.exe)
	cp $(TOP)/src/parse.y .
	./lemon$(B.exe) $(OPT_FEATURE_FLAGS) $(OPTS) -S parse.y
```

We reproduce these recipes directly rather than configuring/building all of
SQLite. The selected configuration is:

| Setting | Resolved value |
| --- | --- |
| Host platform | Linux x86_64 |
| `B.cc` | `cc` (Debian GCC `12.2.0-14+deb12u1`, version 12.2.0) |
| Compiler flags | None; no `CFLAGS`/`CPPFLAGS` expansion |
| `B.exe` | Empty |
| `OPT_FEATURE_FLAGS` | Empty |
| `OPTS` | Empty |
| Lemon arguments | `-S parse.y` |
| Template | Pristine `lempar.c` in the working directory; implicit upstream lookup |
| Input path | Literal relative `parse.y` |
| Output paths | Relative `parse.c`, `parse.h`, `parse.out`, `parse.sql` |
| Working directory | Fresh directory containing `lemon`, `lempar.c`, `parse.y` |
| Environment | `LC_ALL=C`, `TZ=UTC` |

Exact commands, with `ROOT` denoting this repository's absolute path and the
working directory set to the fresh generation directory:

```sh
cc -o lemon "$ROOT/reference/sqlite/tool/lemon.c"
cp "$ROOT/reference/sqlite/tool/lempar.c" .
cp "$ROOT/reference/sqlite/src/parse.y" .
./lemon -S parse.y
```

`scripts/generate-reference.sh` performs these steps (passing an absolute
output path to `cc` is equivalent to the relative `-o lemon` above). `CC` may
select a single compiler executable/path; it is not a shell command or a flag
list. The script intentionally does not consume ambient build/feature flags.
Each run records compiler/platform details and commands in `environment.txt`
and retains build stdout/stderr as well as generation stdout/stderr/status.
This is the configuration's C baseline, not proof of all SQLite feature
combinations or cross-platform/compiler reproducibility.

`-S` produces SQL describing the grammar in addition to the normal C, header,
and state report. It does **not** suppress `parse.out`. All four files are
therefore golden artifacts. Both future generators must use the same relative
paths; no rewriting of `#line` directives or other output normalization is
permitted. Absolute temporary-directory names do not appear in these outputs.

One pinned-source output convention is worth preserving explicitly: with
`-S`, generated `parse.c` contains `#line` references to `parse.sql` as well as
`parse.y`. `ReportTable` opens `.sql` through `file_open`, which changes the
shared `lemp->outname` used for generated line directives. Do not rewrite those
references to `parse.c` in the port or in the comparison scripts.

## Reproduction and comparison

From the repository root, with Bash, a host C compiler, GNU coreutils, and
GNU diffutils installed:

```sh
scripts/test-reference.sh
```

This offline command validates source/golden hashes, independently compiles
and executes C Lemon twice in isolated directories, compares each run against
the committed goldens, and compares the runs to each other. Every generated
artifact must match under both byte comparison and `diff -bw`. Stdout and
stderr must be empty; the captured exit status must be `0` followed by a
newline. Missing prerequisites fail explicitly rather than skipping checks.
On success the temporary builds are deleted. On failure their full contents
are retained and their location and reproduction command are printed.

To keep a generated reference build for inspection, pass a **new**, nonexistent
directory whose parent already exists:

```sh
scripts/generate-reference.sh /tmp/lemon-inspect
scripts/compare-artifacts.sh testdata/golden/sqlite /tmp/lemon-inspect
```

The comparator is deliberately strict for Milestone A: a whitespace-only
change still fails byte comparison, even if its separately reported `diff -bw`
comparison passes. It checks the four generated files and the three captured
process results, not compiler binaries, timestamps, or environment logs.
This is reference reproducibility tooling, not yet the C-versus-Go harness.

Ordinary Go validation requires only the module's Go toolchain, no C compiler
or network:

```sh
GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 go vet ./...
```

The reference Go test checks checked-in source and golden hashes. Milestone B's
grammar tests additionally compare the Go command against checked-in C grammar
expectations. Neither ordinary test executes C or generates a C parser.

## Milestone B grammar oracle

The pin, build configuration, and final-artifact goldens above remain unchanged.
`reference/grammar-snapshot.c` is a separate instrumentation driver that includes
pristine `sqlite/tool/lemon.c` with a renamed entry point. It repeats upstream
initialization and symbol/rule indexing, calls `FindRulePrecedences`, and dumps
grammar metadata before automaton construction. The pinned source is not edited.

`scripts/test-grammar.sh` builds that driver and an uninstrumented C executable
with `cc -o EXECUTABLE SOURCE` (no additional compiler flags), plus the Go command
with `CGO_ENABLED=0`. It sets `LC_ALL=C`, `TZ=UTC`, `GOTOOLCHAIN=local`,
`GOPROXY=off`, and `GOSUMDB=off`. Each case uses isolated directories, the same
literal relative filename `grammar.y`, and executable argv[0] `./lemon`.
The manifest records all options in order. Failures retain compiler/platform/Go
versions, build commands, the case manifest, binaries, inputs, and process outputs.

Snapshots use the instrumented driver; preprocessing/reprints/CLI diagnostics
use pristine C. These are intermediate grammar comparisons, **not** replacements
for the uninstrumented `-S parse.y` final-artifact baseline. Their fixtures,
snapshot schema, and explicit regeneration command are documented in
[`testdata/grammar/README.md`](../testdata/grammar/README.md).

## Golden artifact hashes

`testdata/golden/sqlite/SHA256SUMS` records all expected outputs:

| Artifact | SHA-256 |
| --- | --- |
| `parse.c` | `77ff8fda334c8b0cc01f5734765f994417ba81417d50dedf3d0d7241fe8a4a52` |
| `parse.h` | `bf8058d86e0b5fb75168e288a35c88a651fe7ec4fe43bfdaccc3dae9d4e06674` |
| `parse.out` | `5630aa0d11d56ccf3fba595763b1b59e79e31cebc2e5ae5274aec94469181f0b` |
| `parse.sql` | `f5c5641931f50cb9fbd34b1fdb8cda3921b8d90167d5e4ee29b37ff5664bea0e` |
| `stdout.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `stderr.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `exit-status.txt` | `9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa` |
