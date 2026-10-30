# Command-line reference

`lemon` is a Go executable that reads a Lemon grammar and generates a **C**
parser. This reference describes the implemented command, based on SQLite
3.50.4 Lemon. Verification results and exclusions live in
[the compatibility evidence](../COMPATIBILITY.md); the shorter
[compatibility reference](compatibility.md) summarizes the supported boundary.

## Synopsis

```text
lemon [options] grammar.y
lemon -x
```

Exactly one filename is required, except with `-x`. Options may precede or
follow it. Relative paths are resolved from the working directory. Ordinary
generation writes [parser artifacts](output-files.md); Lemon does not compile
the generated C or supply a tokenizer.

```sh
lemon calculator.y
lemon -q -dout -Ttemplates/lempar.c calculator.y
lemon -DFEATURE -E calculator.y
```

In the second example, `out` must already exist. The third prints preprocessed
input and does not generate a parser.

## Argument syntax

This is the pinned Lemon argument syntax, not Go's standard flag syntax.

| Form | Meaning |
| --- | --- |
| `-q`, `+q` | Set or clear a boolean option. Defaults are clear. Later occurrences win. |
| `-Tfile`, `-dout`, `-DNAME` | String arguments attach directly to the option letter; do **not** insert a space. |
| `+Tfile`, `+DNAME` | String callbacks still receive the suffix; `+` does not undo a string option. Use `-U` to undefine a macro. |
| `D=NAME`, `T=file` | Bare `letter=value` also supplies a string argument. Prefer the attached `-` forms. |
| `g=1` | Invalid: a boolean option cannot use the bare assignment form. |
| `-gtrailing`, `-g=value` | Set `g`: boolean option recognition accepts trailing text, not bundled options. |
| `--` | An undefined option, **not** an end-of-options delimiter. |

For example, `-T lempar.c grammar.y` supplies an empty template argument and
two positional arguments; it is not equivalent to `-Tlempar.c grammar.y`.
There is no conventional `--help` or `-h` option. An unknown option prints a
syntax diagnostic and the upstream-style valid-option list to stderr, then
returns status 1. Project snapshot options below are accepted separately and
are not included in that upstream-style list.

## Options

All options in this table are implemented. The evidence document distinguishes
the differential coverage from untested filesystem and platform boundaries.

| Option | Effect |
| --- | --- |
| `-b` | Show only basis items, rather than full closures, in `.out`. Does not change the automaton. |
| `-c` | Disable default-reduction and SHIFTREDUCE compression. Packed table construction still runs. |
| `-dDIR` | Write artifacts under `DIR`, using the grammar basename. Does not create directories. Without this option, names follow the input pathname, despite the help text's “Default '.'”. |
| `-DNAME` | Define a name for the [grammar preprocessor](directives.md#conditional-preprocessing). `-DNAME=value` defines only `NAME`; the value is discarded. |
| `-E` | Print the preprocessed input and an additional newline to stdout, then stop before grammar parsing. |
| `-fTEXT` | Ignored compiler-option placeholder; empty suffix allowed. |
| `-g` | Reprint the parsed grammar without actions to stdout; skip automaton and artifact generation. Not a lossless grammar/declaration round trip. |
| `-ITEXT` | Ignored compiler include-option placeholder; **not** a grammar include path. Empty suffix allowed. |
| `-m` | Emit makeheaders-style interface sections and a header include in `.c`; suppress Lemon's `.h` generation. Does not execute makeheaders. |
| `-l` | Suppress generated `#line` directives. |
| `-OTEXT` | Ignored compiler optimization-option placeholder; empty suffix allowed. |
| `-p` | Include actions discarded by precedence in `.out`; does not change conflict resolution. |
| `-q` | Suppress `.out`, not `.c`, `.h`, diagnostics, or requested statistics. |
| `-r` | Skip `ResortStates`, including its initialization work. **Pinned quirk:** zero states are retained; this is not a safe “same parser, unsorted” mode. |
| `-s` | Print parser statistics to stdout. With `-g`, machine/table statistics remain unbuilt. |
| `-S` | Also write `.sql` containing symbol/rule/RHS metadata, not packed action tables. Can change `#line` restoration paths in `.c`; see [output files](output-files.md#line-directives-and-reproducibility). |
| `-x` | Print exactly `Lemon version 1.0` and exit successfully, without requiring a filename. Does not identify the project release or SQLite pin. Invalid options are still rejected first. |
| `-TFILE` | Use exactly this parser template. Missing/unreadable explicit templates fail; no embedded fallback is attempted for them. |
| `-UNAME` | Remove the first matching previously defined name. Command-line order matters. |
| `-WTEXT` | Ignored compiler warning-option placeholder; empty suffix allowed. |

Definitions are not deduplicated. Thus `-DA -DA -UA` leaves one definition of
`A`; `-UA -DA` also leaves `A` defined. A removal swaps the last definition into
the removed entry before the remaining list is sorted for processing.
Go safely ignores bare placeholder assignments such as `f=text`; the pinned C
implementation may call a null callback for these spellings. C crash behavior
is not a compatibility requirement.

## Project-only inspection options

These exact long options are **extensions**, not SQLite Lemon options. They
print deterministic implementation snapshots to stdout, not generated C or a
replacement for `.out`.

| Option | Stopping point and flags |
| --- | --- |
| `--grammar-snapshot` | After grammar preparation and inferred rule precedence. Overrides `-g`; `-s` may append statistics. |
| `--automaton-snapshot` | After FIRST/lookahead/state construction and conflict resolution, before table optimization. Overrides the grammar snapshot and `-g`. Ignores generation/report flags, including `-s`. |
| `--table-snapshot` | After compression, state renumbering, and packing. Honors `-c` and `-r`; otherwise ignores generation/report flags, including `-s`. Overrides other snapshots and `-g`. |

`-x` and `-E` exit before any snapshot mode. Automaton/table modes print their
snapshot even when automaton diagnostics or unresolved conflicts occur, then
return 1. Initial grammar/preprocessor failures stop before a snapshot.
Snapshot field layouts are documented for maintainers in the
[grammar](../../testdata/grammar/README.md),
[automaton](../../testdata/automaton/README.md), and
[table](../../testdata/tables/README.md) fixture guides.

## Streams and exit status

| Result | stdout | stderr | Status |
| --- | --- | --- | --- |
| Ordinary success | Empty unless `-s` | Empty | 0 |
| `-x`, `-E`, `-g`, snapshot success | Requested text; `-s` only where applicable | Empty | 0 |
| Option/argument, input, or preprocessor failure | No successful result promised | Diagnostic | Nonzero |
| Early grammar errors | No parser | Grammar diagnostics | Grammar error count (not necessarily 1) |
| Generation errors or unresolved conflicts | Requested statistics may still appear | Diagnostics/conflict count | 1 |

Files can exist after failure, and suppressed artifacts from previous runs are
not deleted. Do not use file presence as the success check.

## Related documentation

- [Grammar syntax](grammar.md) and [declarations](directives.md)
- [Output files and template selection](output-files.md)
- [Generate a parser](../how-to/generate-parser.md)
- [Troubleshoot generation](../how-to/troubleshoot.md)
- [Documentation index](../README.md)
