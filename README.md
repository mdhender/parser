# parser

A faithful Go port of SQLite's Lemon parser generator, currently at
**Milestone C: parser automaton**. The Go command preprocesses, scans, and
parses Lemon grammars, computes FIRST/nullable sets, constructs LALR states and
lookaheads, and resolves conflicts, including for SQLite's full grammar. Packed
tables and C parser generation are not implemented yet; normal generation
fails explicitly without creating output files. The eventual executable will
generate C parsers, not Go parsers.

The reference is pinned to SQLite 3.50.4. Pristine sources and golden outputs
are committed; tests do not download anything or update fixtures.

## Validate

Offline grammar and fixture-integrity tests (Go version from `go.mod`; no C compiler):

```sh
GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 go vet ./...
```

Independent C reference rebuilds and byte-for-byte golden comparisons
(Bash, `cc`, GNU coreutils and diffutils):

```sh
scripts/test-reference.sh
```

Grammar-processing C-versus-Go comparisons, including internal snapshots,
stdout, stderr, exit status, and output-file inventories:

```sh
scripts/test-grammar.sh
scripts/test-automaton.sh
```

This explicit differential command requires a C compiler, Go, Bash, coreutils,
and diffutils. Both byte comparison and `diff -bw` must pass; failures retain
their full inputs and outputs. Ordinary Go tests use checked-in C expectations
and never execute C or download dependencies.

## Use the implemented stages

```sh
CGO_ENABLED=0 go build -o /tmp/lemon ./cmd/lemon
/tmp/lemon -E -DSQLITE_OMIT_TRIGGER reference/sqlite/src/parse.y
/tmp/lemon -g reference/sqlite/src/parse.y
/tmp/lemon --grammar-snapshot reference/sqlite/src/parse.y
/tmp/lemon --automaton-snapshot reference/sqlite/src/parse.y
```

`-E` prints preprocessed input; `-g` reprints rules without actions in upstream
order. `--grammar-snapshot` is a project-only inspection option: it emits
deterministic symbol/rule/declaration metadata after precedence assignment.
See [snapshot format and fixture commands](testdata/grammar/README.md).
`--automaton-snapshot` is a separate inspection mode that stops after conflict
resolution, before compression or state renumbering. It emits discovery-order
states, configurations, lookaheads, actions, and FIRST/nullable sets. Conflicts
and automaton errors diagnose and return status 1, while retaining the snapshot.
See [automaton fixtures and format](testdata/automaton/README.md).

See [reference provenance and reproduction](docs/REFERENCE.md),
[architecture](docs/ARCHITECTURE.md), [porting log](docs/PORTING.md), and
[compatibility scope](docs/COMPATIBILITY.md). Work stops at Milestone C pending
authorization for Milestone D.
