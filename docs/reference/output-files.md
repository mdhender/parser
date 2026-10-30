# Output files and parser API reference

Ordinary Lemon generation writes a C parser and associated files. It does not
generate a Go parser, compile C, run makeheaders, or create output directories.
[CLI modes](cli.md) such as `-E`, `-g`, and project snapshots stop without
generating these artifacts.

## Artifact inventory

For input `calculator.y`, ordinary successful generation produces:

| Artifact | When written | Contents |
| --- | --- | --- |
| `calculator.c` | Ordinary generation | Parser template with token constants, semantic union, packed tables, actions, destructors, and hooks inserted. C runtime/API depend on the selected template and grammar. |
| `calculator.h` | Unless `-m` | Terminal `#define` constants, using `%token_prefix` if set. Not a complete parser API header. EOF is code 0 and has no ordinary token definition here. |
| `calculator.out` | Unless `-q` | Human-readable automaton/action report, symbol information, and rules. `-b` limits state items to basis items; `-p` shows precedence-discarded actions. |
| `calculator.sql` | Only with `-S` | SQL transaction defining/inserting `symbol`, `rule`, and `rulerhs` grammar metadata. Not the packed parser action tables. |

With `-m`, `.c` includes makeheaders-compatible interface sections and a header
include, but Lemon does not generate `.h` or execute makeheaders. Neither `-m`
nor `-q` deletes a previously generated file. An existing `.h` whose complete
contents already match is left untouched, preserving its timestamp; stale
contents are replaced.

The generator may write files despite unresolved conflicts or later emission
errors. Some failures leave empty or partial files, and old files may remain.
Always check the [exit status](cli.md#streams-and-exit-status).

## Generated C API

The **selected template dictates the API**. With the default pinned `lempar.c`,
no `%name`, no extra argument/context, and default `YYMALLOCARGTYPE` (`size_t`),
the principal signatures are:

| Signature | Purpose |
| --- | --- |
| `void *ParseAlloc(void *(*mallocProc)(size_t));` | Allocate and initialize a parser; returns NULL if allocation fails. |
| `void Parse(void *parser, int token, ParseTOKENTYPE value);` | Push one token and its semantic value. Returns no parse status; use grammar hooks for acceptance/errors. |
| `void ParseFree(void *parser, void (*freeProc)(void *));` | Finalize the parser, then release the parser object with the supplied deallocator. |
| `void ParseInit(void *parser);` | Initialize caller-allocated parser storage, whose layout/size must be available from generated source. |
| `void ParseFinalize(void *parser);` | Destroy remaining stack values and release grown stack storage, but not the parser object. |

`ParseTOKENTYPE` is generated from `%token_type` (default `void*`). Positive token
codes come from `.h`; **token 0 signals end of input**, not an ordinary token.
Pass a value of the declared token type for that call as well. A typical heap
lifecycle is `ParseAlloc`, repeated `Parse` calls, a `Parse` call with token 0,
then `ParseFree`. Caller-owned storage instead uses `ParseInit`/`ParseFinalize`;
`Parse_ENGINEALWAYSONSTACK` omits the allocation/free entry points.

`%name Calculator` changes these names to `Calculator`, `CalculatorAlloc`, and
so on. `%extra_argument {DECL}` appends a parameter to each `Parse` call;
`%extra_context {DECL}` appends one to `ParseAlloc` and `ParseInit`, not
`ParseFree` or `ParseFinalize`. These declarations therefore change the
signatures above. The ordinary generated `.h` contains **only token constants**,
not these function prototypes; the application must supply matching
declarations or otherwise expose the generated implementation.

## Naming and directories

Without `-d`, output follows the grammar pathname. With `-dDIR`, the input
basename is placed under `DIR`. Paths are relative to the invocation's working
directory unless absolute.

| Input/options | C output | Other suffixes |
| --- | --- | --- |
| `calculator.y` | `calculator.c` | Same stem with `.h`, `.out`, optional `.sql`. |
| `grammars/calculator.y` | `grammars/calculator.c` | Same directory/stem. |
| `-dbuild grammars/calculator.y` | `build/calculator.c` | Same output directory/stem. |
| `calculator` | `calculator.c` | Suffix appended if no dot exists in the constructed pathname. |

The implementation removes everything from the **last dot in the constructed
pathname**, not specifically the extension of the basename, then appends the
suffix. This pinned behavior can surprise for extensionless files below dotted
directories. `/` is the directory separator used by these naming routines.
The help text's “Default '.'” does not override the actual input-path behavior.

## Parser template selection

Selection order is significant:

| Priority | Candidate |
| --- | --- |
| 1 | Explicit `-TFILE`. This is an override, not an additional search directory. Failure is an error. |
| 2 | Grammar-adjacent `.lt` file: for `grammars/calculator.y`, `grammars/calculator.lt`. |
| 3 | `lempar.c` in the working directory. |
| 4 | If argv[0] contains `/`, `lempar.c` beside that invocation path; otherwise search `PATH` in order for `lempar.c`. |
| 5 | **Project extension:** embedded, unchanged pinned SQLite 3.50.4 `lempar.c`, when the final external candidate is absent. |

The embedded fallback is implemented, not planned: the standalone Go binary
can generate without an installed template. It does not override external
files, repair malformed templates, or mask explicit `-T` failures or read errors
on selected external files. Early grammar-adjacent/current candidates are
probed for readability before later lookup. A slash-containing invocation does
not additionally search `PATH`. If `PATH` is unset, the search default is
`.:/bin:/usr/bin`.

Template chunks are copied between `%%` markers and filled in a fixed sequence.
`%name` rewrites eligible `Parse` prefixes while copying. Arbitrary C files are
not interchangeable templates; no general template validation guarantees that
a selected file produces a compilable parser. See
[custom templates](../how-to/custom-template.md) and
[parser template explanation](../explanation/parser-templates.md).

## Generated data and identifiers

| Item | Reference behavior |
| --- | --- |
| Token numbers | First-seen terminal ordering, controllable with `%token`; EOF is 0. Keep scanner token constants synchronized with `.h`. |
| Semantic storage | Shared terminal type and nonterminal type slots form the generated semantic union. |
| Rule IDs | Rules with actions are assigned emitted IDs before rules without actions; original grammar order and emitted numbering need not agree. |
| State IDs | Discovery order and optimized/renumbered order differ. `-r` preserves a pinned zero-retained-state bug, not a usable unsorted table mode. |
| Packed arrays | `.c` includes `yy_action`, `yy_lookahead`, `yy_shift_ofst`, `yy_reduce_ofst`, and `yy_default`. `-c` disables compression, not packing. |
| SQL RHS positions | A multiterminal produces multiple `rulerhs` rows at one position, one per constituent. |

The `.out` report and snapshot extensions serve different purposes. Snapshots
expose internal stage representations; they are stdout inspection output, not
additional parser files.

## Line directives and reproducibility

Generated text normally contains `#line` directives pointing into the grammar
and restoring the output filename. `-l` suppresses them. Input path spelling,
working directory, output paths, definition list/order, template selection,
and generation flags can therefore affect bytes even for equivalent grammar
rules. The C prologue records macro names actually used during conditional
evaluation; short-circuited names may not appear.

**Pinned `-S` quirk:** opening `.sql` changes a shared output-name field. Later
line restorations in `.c` can name the `.sql` path. This behavior is preserved;
do not normalize it away when comparing output.

The [compatibility evidence](../COMPATIBILITY.md) records exact artifact and
process-result comparisons. The primary SQLite baseline generates `.c`, `.h`,
`.out`, and `.sql` with `./lemon -S parse.y`; reproducible comparisons use
equivalent input pathnames and options in separate directories.

## Related documentation

- [CLI](cli.md), [declarations](directives.md), [compatibility](compatibility.md)
- [Generate a parser](../how-to/generate-parser.md)
- [First parser tutorial](../tutorials/first-parser.md)
- [Documentation index](../README.md)
