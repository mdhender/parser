# Agent instructions

## Project and scope

This repository will contain a standalone Go port of SQLite's Lemon parser
generator. The Go executable must generate C parsers, not Go parsers.

Read `port-prompt.md` before implementation work. It is the detailed specification
for compatibility, milestones, reference fixtures, and documentation. The current
implementation authorization covers **Milestones A through E**: establish a
reproducible C reference baseline, port grammar processing, construct the
parser automaton, generate packed tables, and emit C parser artifacts. Stop after
Milestone E and await authorization before final compatibility (Milestone F).
The initial Milestone A limit in `port-prompt.md` was extended by the owner's
explicit requests to implement Milestones B, C, D, and E.

The repository contains the pinned C baseline and Go grammar, automaton, table,
and C emission implementation in `internal/lemon`, the `cmd/lemon` command,
offline Go tests, and explicit C reference/differential scripts. See
`docs/PORTING.md` for the verified boundary.
The layout proposed in `port-prompt.md` is guidance rather than a required file
layout.

## Implementation principles

- Use only the Go standard library and built-in testing package. The finished Go
  executable must not require cgo, a C compiler, or external Go dependencies.
- SQLite's official C implementation is authoritative. Do not use another Go port
  as an implementation source or substitute different parsing algorithms.
- Preserve algorithms, data flow, ordering, command-line behavior, diagnostics,
  and output conventions. Keep translated structures and routines recognizable
  relative to the C source.
- Prefer explicit code and clear subsystem boundaries over interfaces,
  reflection, goroutines, or unnecessary abstractions. Use conventional Go naming
  and `gofmt` where they do not obscure source comparison.
- Do not rely on Go map iteration for deterministic output. Pay particular
  attention to pointer identity, integer conversions, linked-list traversal,
  comparators, sorting, character classification, and output formatting.
- Translate incrementally and verify each subsystem. Diagnose the earliest
  divergence from the C reference rather than patching generated output. Never
  hardcode SQLite's grammar, symbol numbers, tables, or expected artifacts.

## Reference and compatibility

- Pin `tool/lemon.c`, `tool/lempar.c`, and `src/parse.y` from the same immutable
  revision of the official SQLite repository, <https://sqlite.org/src>. Record
  the revision and SHA-256 hashes in `docs/REFERENCE.md`. Do not silently update it.
- Keep pinned sources pristine. Instrument separate copies only; use an
  uninstrumented reference executable for final comparisons.
- Record SQLite's exact parser-generation invocation and build configuration,
  including options, feature flags, template, paths, working directory, compiler,
  platform, and relevant environment. Preserve reproducible scripts and golden
  artifacts. Rebuilding the C reference must reproduce its goldens byte for byte
  before Go translation starts.
- Run C and Go comparisons in isolated temporary directories with equivalent
  input paths and options. Compare stdout, stderr, exit status, and every generated
  artifact, including malformed grammars and invalid options. Retain full failure
  artifacts and make failures reproducible.
- Compare artifacts both byte for byte and with `diff -bw`. The primary acceptance
  fixture is SQLite's `parse.c`; also compare `parse.h` and `parse.out` when
  produced. Review and document every byte-level difference, even if `diff -bw`
  passes: whitespace changes can alter C strings or preprocessor definitions.
  Do not normalize or reorder output to hide differences.

## Validation

Use the toolchain version declared in `go.mod`. Orb provisioning is in
`.agents/setup`; `.agents/resume` checks the installed toolchain on wake.

For Go implementation changes, format changed Go files and run:

```sh
go test ./...
go vet ./...
```

Before Go packages exist, these commands may report that no packages match; this
is not evidence of a passing test suite. Do not add placeholder code or tests just
to make the commands succeed.

Ordinary `go test ./...` must run offline without a C compiler, using checked-in
fixtures. Reference reproducibility and C-versus-Go differential checks must have
a separate documented, explicit command that fails clearly when required tools
are missing. Tests must not download reference sources or dependencies.

Once generation is implemented, repeat tests to detect nondeterminism and cover
small grammars, precedence, fallback and wildcard tokens, error recovery,
directives, malformed inputs, and supported command-line options. Compilation or
runtime testing of generated C supplements, but never replaces, artifact
comparisons. Do not claim compatibility for behavior not verified against the
reference.

## Documentation and delivery

As milestones are implemented, maintain the documents required by
`port-prompt.md`: `docs/REFERENCE.md`, `docs/ARCHITECTURE.md`, `docs/PORTING.md`, and
`docs/COMPATIBILITY.md`. Record translated C routines, Go files, checks, results,
intentional deviations, unresolved differences, and the next recommended task.
Distinguish implemented behavior from reference-verified compatibility.

Work directly on `main`; do not create branches. The owner authorizes commits and
pushes to `main` when all applicable tests are green. Make small, reviewable
commits with messages identifying the subsystem or milestone. Preserve unrelated
worktree changes. Report checks honestly, including when no tests exist, and
state whether changes are local, committed, or pushed.

Do not create GitHub CI actions or workflows.
