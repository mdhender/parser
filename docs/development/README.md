# Development and maintainer records

For using Lemon, start at the [user documentation index](../README.md). This
area is for maintaining the Go port and reproducing its evidence.

## Engineering records

The existing records remain at their original paths to preserve links from
source, scripts, fixtures, commits, and issue discussions. They belong to the
maintainer documentation, not to the user-facing reference:

- [Pinned reference provenance and reproduction](../REFERENCE.md)
- [Implementation architecture and C-to-Go mapping](../ARCHITECTURE.md)
- [Milestone porting log](../PORTING.md)
- [Detailed compatibility evidence and exclusions](../COMPATIBILITY.md)
- [Port specification](../../port-prompt.md) and [agent/contributor constraints](../../AGENTS.md)
- [Release history](../../CHANGELOG.md)

## Go checks

Use the toolchain in [`go.mod`](../../go.mod). The generator executable has no
external dependencies; the repository root's version API uses a semver module.
If its module is not cached, fetch it once before offline testing:

```sh
go mod download
```

Then run from the repository root (no C compiler or network required):

```sh
GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 go vet ./...
```

## Documentation examples

With Go, Bash, a C compiler named `cc`, awk, and standard shell utilities:

```sh
scripts/test-docs.sh
```

This executes the tutorial's actual shell block, generates with the embedded
template in a new temporary directory, compiles the supplied C driver, and
requires output exactly `Result: 7` followed by a newline. It is separate from
offline Go tests so those tests never require a C compiler.

The Go calculator tutorial is extracted, generated, and executed by
`TestGoTutorial` in the offline Go suite, requiring exactly `Result: 11`.
`TestGoParsers` builds isolated dependency-free modules and executes generated
parsers with and without compression; `TestGoCommands` builds/invokes both names.

`go test ./internal/ll1` checks LL(1) analysis and the `glime` command, then
compiles and executes both stack and recursive standalone parsers, including the actual
LL(1) tutorial grammar, nested completion order, heterogeneous values, epsilon,
deep nesting, and malformed inputs. No C compiler is needed.

The [PL/0 example](../../examples/pl0/README.md) keeps its generated LALR and
LL(1) parsers checked in. `go test ./examples/pl0/...` regenerates into temporary
directories and requires exact bytes, compares all three frontends' shared ASTs,
and runs them through the same interpreter against independent expectations.
After changing the example grammars, run `go generate ./examples/pl0/...`.
The example's runtime imports no generator internals; generator APIs are used
only by its regeneration tests. These checks are part of ordinary offline tests.
`TestPL0Tutorials` executes every shell block from all three PL/0 tutorials in
disposable source copies, including generation, the initial CLI example, and
the complete Go callers. It requires the exact documented AST summary and result.
The recursive package generates from `ll1/grammar.y`, not a second predictive
grammar. Recursive depth tests cover 4,096 levels and do not establish unlimited
Go call-stack depth. See [the recursive tutorial](../tutorials/pl0-recursive.md).

For an explicit generated-Go versus pristine generated-C runtime comparison:

```sh
scripts/test-go-runtime.sh
```

This separate C-backed check compares observable results, hooks, and destructor
effects for equivalent Go/C grammars, for both command names and compression
modes. It does not compare generated Go source to C source.

## Explicit C reference comparisons

These commands need Bash, `cc` (or `CC`), GNU coreutils/diffutils, and Go for
C-versus-Go checks. They do not download sources or refresh goldens:

```sh
scripts/test-reference.sh
scripts/test-grammar.sh
scripts/test-automaton.sh
scripts/test-tables.sh
scripts/test-generation.sh
```

They compare stdout, stderr, exit status, file inventories, exact bytes, and
`diff -bw`, retaining failure artifacts and reproduction commands. See the
[grammar](../../testdata/grammar/README.md),
[automaton](../../testdata/automaton/README.md),
[table](../../testdata/tables/README.md), and
[generation](../../testdata/generation/README.md) fixture guides for modes and schemas.
Do not update expectations to hide a Go/C divergence.

The full acceptance command is **Linux ELF-specific** and additionally needs
`readelf` from binutils. It runs all the above checks, repeats Go tests, verifies
a cgo-disabled binary, and compares two standalone runs with no tools on PATH:

```sh
scripts/test-compatibility.sh
```

## Optional SQLite source-tree integration

Provide a local, hash-verified official SQLite 3.50.4 archive; these commands do
not download it. Additional requirements include make, unzip, and Tcl runtime
and development files:

```sh
scripts/test-compatibility.sh /absolute/path/to/sqlite-src-3500400.zip
# Or run only the supplementary integration:
scripts/test-sqlite.sh /absolute/path/to/sqlite-src-3500400.zip
```

The [SQLite integration guide](../../testdata/sqlite/README.md) records archive
provenance, build configuration, selected regressions, and failure reproduction.
Compilation/runtime checks supplement artifact comparisons; they do not replace
them or establish exhaustive feature/platform compatibility.
