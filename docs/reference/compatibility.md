# Compatibility reference

This project is a Go port of SQLite's Lemon **C parser generator**, not a Go
parser generator or a replacement grammar dialect. Implemented behavior and
executed compatibility checks are distinct. This page states the user-facing
boundary; detailed commands, matrices, results, and engineering exclusions
remain in [the existing compatibility evidence](../COMPATIBILITY.md).

## Authoritative reference

| Item | Pin/contract |
| --- | --- |
| SQLite version | **3.50.4** |
| Fossil revision | `4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3` |
| Authoritative inputs | Pinned `tool/lemon.c`, `tool/lempar.c`, and `src/parse.y` from that same revision. |
| Source provenance/hashes | [Reference baseline](../REFERENCE.md). |
| Primary acceptance case | Equivalent isolated invocation `./lemon -S parse.y`, with the pinned template/configuration. |
| Comparison contract | stdout, stderr, exit status, complete artifact inventory, exact bytes, and `diff -bw`; no output normalization. |
| Current recorded result | Milestone F acceptance complete; all four pinned baseline artifacts match byte for byte, with no unexplained differences in executed comparisons. |

`lemon -x` prints the inherited string `Lemon version 1.0`; it does **not**
identify this source revision or the project release. A different external
template can change generated runtime behavior even when the generator binary
is unchanged.

## Supported surface

| Surface | Implemented behavior and verification boundary |
| --- | --- |
| [CLI](cli.md) | Complete pinned option table and argument quirks; grammar/preprocessing, generation, report and explicit template/output modes have differential coverage. |
| [Grammar syntax](grammar.md) | Rules, aliases, empty productions/actions, precedence, named/inline multiterminals, fallback, wildcard and `error`; parser and later-stage effects have separate fixture coverage. |
| [Declarations](directives.md) | Every keyword in the pinned declaration dispatch is implemented and snapshot-verified; generated effects are covered by generation fixtures and SQLite variants, not every possible combination. |
| [Conditional preprocessing](directives.md#conditional-preprocessing) | `%if`, `%ifdef`, `%ifndef`, `%else`, `%endif`, CLI definitions/undefinitions, nesting/exclusion and pinned evaluator quirks. |
| [Artifacts](output-files.md) | C source, token header, automaton report, optional SQL metadata; byte comparisons include files produced on errors. |
| Generated runtime | Selected arithmetic, fallback/wildcard, recovery, destruction, dynamic-stack and fixed-stack scenarios, with/without assertions and selected compression configurations. Not exhaustive C runtime verification. |
| SQLite integration | Recorded pinned source-tree build and selected upstream regression campaign. Not the complete SQLite release test suite or every feature combination. |

Generating a C file that compiles is supplementary evidence, not a substitute
for artifact comparison. SQLite's `parse.c` needs its matching SQLite source
tree, headers, types, and compile flags; it is not a standalone generic parser.

## Intentional project extensions

| Extension | Difference from pinned C Lemon |
| --- | --- |
| Embedded default template | The binary embeds the unchanged pinned `lempar.c` and uses it after the final external candidate is absent. External precedence is preserved; explicit overrides and selected-file read errors are not masked. No separately installed template is required for normal standalone use. |
| `--grammar-snapshot` | Inspection output after grammar preparation/inferred precedence. Not an upstream flag or a generated parser artifact. |
| `--automaton-snapshot` | Inspection output after conflict resolution. Preserves snapshots on automaton errors/conflicts, returning 1. |
| `--table-snapshot` | Inspection output after table compression/renumbering/packing. Honors `-c`/`-r`; does not perform emission-time action rewriting. |

See [snapshot precedence and flag handling](cli.md#project-only-inspection-options).
These extensions do not imply that C Lemon recognizes those long options or
has an embedded fallback. Internal state is per invocation in Go rather than
the C process-static pools; deterministic observable order is preserved for
the verified cases.

## Preserved quirks

| Behavior | Consequence |
| --- | --- |
| `-r` skips initialization as well as sorting | Zero retained states in the pinned cases; do not use it expecting an otherwise equivalent unsorted parser. |
| Boolean options accept trailing text; `+` clears flags; `--` is invalid | This is not a conventional modern flag parser. String values attach to letters. |
| No `-d` means input-path-based output | Output need not be in the current directory, regardless of the help text. |
| `-S` changes the shared current output name | `.c` line-restoration directives can refer to `.sql`. |
| Preprocessor short-circuits the remaining expression | Mixed `&&`/`\|\|` does not follow ordinary C precedence. Use explicit parentheses. |
| Preprocessing precedes comment/code scanning | Column-one directive-looking lines inside C text are still processed. |
| Missing final grammar-state validation | Some incomplete EOF constructs are accepted. Not every malformed grammar is diagnosed. |
| Files may be produced on failure | Check status; file presence is not success. Suppressed old artifacts are not deleted. |

## Exclusions and unverified boundaries

- Exact behavior for undefined C operations and crashes is not a requirement.
  Go bounds unfinished code-block scanning at EOF, safely ignores bare
  placeholder callbacks (`f=...`, `I=...`, `O=...`, `W=...`), and does not
  reproduce invalid-start C assertion aborts.
- C duplicate-shift ties spanning allocation blocks depend on undefined pointer
  subtraction. Portable equivalence of their winners is excluded; ordinary
  rule-index tie breaking and within-block fixture behavior are covered.
- Verification is recorded on Linux x86_64 with the fixed C locale. Other C
  locales, platform integer widths, allocator failures, and files changing
  during reads are outside the executed claim.
- Complete template/filesystem/permission-denied behavior, all integer/type-width
  boundaries, every feature combination, and exhaustive malformed input are
  not verified. PATH override precedence has offline Go coverage; full PATH-only
  template search is not a new C differential case.
- Selected generated-runtime and SQLite regressions do not establish every
  template macro configuration, custom template, recovery path, or full SQLite
  feature/platform compatibility.

These limits are not unexplained differences in the recorded successful
comparisons. Consult the [evidence](../COMPATIBILITY.md) before expanding a
compatibility claim or treating untested behavior as a known incompatibility.

## Execution requirements

The standalone command needs no C compiler or cgo to generate parsers; the
embedded template covers missing external defaults. Compiling generated C
requires a C toolchain and the application's headers/runtime. Explicit C oracle
checks also require the documented C/testing tools; ordinary Go tests consume
checked-in fixtures offline rather than compiling the oracle or downloading
sources. The repository's Go toolchain requirement is declared in `go.mod`.

## Related documentation

- [Install](../how-to/install.md) and [generate a parser](../how-to/generate-parser.md)
- [Lemon versus yacc](../explanation/lemon-vs-yacc.md)
- [Maintainer index](../development/README.md)
- [Documentation index](../README.md)
