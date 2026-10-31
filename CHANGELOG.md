# Changelog

## Unreleased

- Add `glime`, a separate LL(1) Go parser generator (issue #14), with
  check/explain/generate commands, deterministic predictive analysis and
  diagnostics, deferred typed stack actions, standalone runtime tests, and
  a calculator tutorial. Reserve recursive descent with an explicit error;
  document unsupported Lemon directives. Bump the feature version to 1.2.0.
- Add the `glcc` and `glemon` Go-parser generators through one shared backend
  (issue #13), with typed actions, heterogeneous values, fallback/wildcard,
  recovery, hooks/destructors, and a self-contained Go runtime. Include command
  and executable runtime tests, C/Go behavioral comparisons, a calculator
  tutorial, and a Go API reference. Keep `lemon`'s C interface and pinned output
  unchanged. Bump the feature version to 1.1.0.
- Organize user documentation by Diátaxis: a runnable first-parser tutorial,
  installation/generation/template/troubleshooting guides, syntax and CLI
  references, and conceptual explanations (issues #6–#12). Keep engineering
  records at stable paths behind a separate development index. Add an explicit
  C-backed documentation example test, separate from offline Go tests.
- Embed the pristine pinned SQLite 3.50.4 parser template as the final fallback
  when no external template is found (issue #1). Preserve external lookup order
  and explicit `-T` errors. No template installation is required for default
  generation; pinned sources and SQLite golden outputs are unchanged.

## v1.0.0 — 2026-10-30

Initial stable release of the standalone Go port of SQLite's Lemon parser
generator. Milestones A through F are complete. The executable generates **C
parsers**, not Go parsers.

- Implements grammar scanning/preprocessing, declarations, precedence, LALR
  automaton construction, conflict resolution, table compression/packing, and
  C/header/report/optional SQL output using the original `lempar.c` template.
- Pins the reference to SQLite **3.50.4**, Fossil check-in
  `4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3`.
  All four baseline artifacts (`parse.c`, `parse.h`, `parse.out`, `parse.sql`)
  match the original C generator byte for byte, with no unexplained differences.
- Includes 196 differential cases, offline golden tests, repeated deterministic
  generation checks, and generated-C runtime checks. The Milestone F integration
  campaign also built SQLite and passed 2,534 selected upstream tests.
- Builds with Go **1.27.3**, the standard library only, and no cgo. Running the
  generator requires neither a C compiler nor a C runtime dependency; the
  parser template remains a runtime input. Supply the pinned template with
  `-Treference/sqlite/tool/lempar.c` or use the documented template search paths.
- Preserves upstream command-line behavior and adds the project-only
  `--grammar-snapshot`, `--automaton-snapshot`, and `--table-snapshot` modes.

Compatibility is scoped to the pinned revision and documented validation matrix
on Linux x86_64, not every Lemon input, SQLite feature combination, platform, or
locale. Undefined C behavior is excluded. See
[COMPATIBILITY.md](docs/COMPATIBILITY.md) for the precise supported boundary and
intentional deviations, and [REFERENCE.md](docs/REFERENCE.md) for provenance.

Release validation: `scripts/test-compatibility.sh` runs repeated offline Go
tests, vet, standalone generation, C reference reproducibility, all differential
cases, and generated-C runtime checks. SQLite source-tree integration is a
separate optional check requiring the locally supplied pinned source archive;
its recorded Milestone F results are not a new release-time integration run.
