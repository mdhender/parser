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
expectations. Neither executes C; Milestone E's ordinary tests additionally
generate C text and compare its inventory/hashes without compiling it.

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

## Milestone C automaton oracle

`reference/automaton-snapshot.c` separately includes pristine `lemon.c`, repeats
the same initialization, and runs `FindRulePrecedences`, `FindFirstSets`,
`FindStates`, `FindLinks`, `FindFollowSets`, and `FindActions`. It stops before
compression/resorting. It prints original symbol/rule/discovery-state numbers,
FIRST/nullable sets, basis and closure configurations, final lookaheads, and all
resolved/conflicting actions. It preserves diagnostics and returns the boolean
error-or-conflict status after automaton construction.

`scripts/test-automaton.sh` uses the same compiler, environment, literal input
path, isolated working directories, byte/`diff -bw` comparisons, and failure
retention policy as the grammar harness. Its manifest records every option;
ordinary Go tests read C-generated saved expectations offline. The instrumented
driver accepts `-D`, `-U`, and `-l`; upstream CLI syntax/errors remain covered by
the grammar harness using pristine C. The [automaton fixture guide](../testdata/automaton/README.md)
describes the schema and explicit regeneration/repetition commands. These
intermediate comparisons do not replace final parser artifact acceptance.
Milestone A's sources, invocation, and artifact hashes below are unchanged.

## Milestone D table oracle

`reference/table-snapshot.c` includes the same pristine pin and continues the
automaton initialization through `CompressTables` (unless `-c`) and
`ResortStates` (unless `-r`). It copies the original `ReportTable` numeric
ranges, axset/acttab packing, and `doesReduce` blocks, then serializes optimized
states/actions, numeric codes, raw entries, offset metadata, and final numeric
arrays without invoking templates or C-code rewriting. The source and template
hashes, selected configuration, and final-artifact goldens remain unchanged.

`scripts/test-tables.sh` builds the driver with `${CC:-cc} -o EXECUTABLE SOURCE`
and Go with `CGO_ENABLED=0`, using the same offline environment, isolated
`grammar.y` paths, `./lemon` name, strict byte and `diff -bw` checks, and failure
retention policy as the earlier harnesses. Its restricted options are `c`, `D`,
`l`, `r`, and `U`; full CLI diagnostics remain tested against pristine C by the
grammar harness. The [table fixture guide](../testdata/tables/README.md) records
the schema, 48-case matrix, C-only regeneration/repetition commands, and counts.

As an independent numeric cross-check, all five arrays prepared by Go for the
default SQLite grammar were compared with their values in the uninstrumented
Milestone A `parse.c` golden: `yy_action` (2,207), `yy_lookahead` (2,394),
`yy_shift_ofst` (583), `yy_reduce_ofst` (413), and `yy_default` (583). Every
integer matched. This comparison extracted only array initializers and removed
C index comments; it did not normalize an artifact comparison or establish
generated-C text compatibility. Milestone E separately verifies full output below.

## Milestone E full-output oracle

The source pin, build configuration, and Milestone A artifact goldens remain
unchanged. `scripts/test-generation.sh` builds pristine `lemon.c` (no snapshot
instrumentation) with the same `cc -o EXECUTABLE SOURCE` command and Go with
`CGO_ENABLED=0`. The environment is `LC_ALL=C`, `TZ=UTC`, `GOTOOLCHAIN=local`,
`GOPROXY=off`, and `GOSUMDB=off`. It records the compiler, platform, Go version,
invocation convention, and manifest in its isolated work directory.

The [generation matrix](../testdata/generation/cases.tsv) gives exact ordered
options and source paths. Each run copies its source to `grammar.y`, except
`sqlite-baseline`, which uses **`./lemon -S parse.y`**, exactly as above.
The template is the pinned `lempar.c`, copied to the current directory, adjacent
`.lt`, or executable directory according to the manifest operation. The output
directory case precreates `out`; `missing` is intentionally absent. C and Go
always receive equivalent paths. All four baseline Go artifacts reproduce the
hashes below exactly, not merely under whitespace-insensitive comparison.

All 52 cases pass exact and `diff -bw` artifact/process comparisons. Compact
C-derived inventories/hashes are checked in under `testdata/generation/golden`;
ordinary Go tests use these offline. Explicit differential runs compare actual
files, not just hashes, and retain full failure artifacts. C-only update/repeat
and reference-only modes are documented in the
[fixture guide](../testdata/generation/README.md). The runtime case additionally
compiles and executes a small arithmetic parser; this does not compile SQLite
or establish its runtime compatibility. Final Milestone F sign-off remains open.

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
