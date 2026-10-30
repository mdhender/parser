# Pinned Lemon compatibility — Milestone B

## Status and pin

**Grammar processing is implemented and reference-verified for the cases
below. C parser generation is not implemented; no parser-output or runtime
compatibility is claimed.** Milestone A's reference baseline remains unchanged.
Milestone B adds the Go executable, offline grammar goldens, and explicit
C-versus-Go differential checks.

Reference: SQLite **3.50.4**, Fossil revision
`4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3`.
Sources are [`lemon.c`](../reference/sqlite/tool/lemon.c) (6,056 lines),
[`lempar.c`](../reference/sqlite/tool/lempar.c), and
[`parse.y`](../reference/sqlite/src/parse.y). `L` ranges below refer to
`lemon.c` unless another file is named. See [REFERENCE.md](REFERENCE.md) for
source provenance, the selected configuration, artifact hashes, and executed
baseline checks. The tested grammar-stage matrix is recorded in
[`testdata/grammar/cases.tsv`](../testdata/grammar/cases.tsv); its expectations
come from pinned C, never from Go output.
See [ARCHITECTURE.md](ARCHITECTURE.md) for the processing pipeline and hazards.

## Executed comparisons and scope

`scripts/test-grammar.sh` rebuilds both C oracles and the Go command offline.
All 58 manifest cases pass both comparisons, including the CLI cases below.
In separate directories each receives the same relative `grammar.y` path and
`./lemon` argv[0]. It checks C against saved goldens and Go against C, comparing
stdout, stderr, shell exit status, and the complete generated-file inventory.
Both strict bytes and `diff -bw` must match, without normalization. Failure
artifacts are retained with a reproduction command. `--reference-only` checks
goldens and repeated C execution without building Go. Ordinary `go test ./...`
uses the same checked-in expectations without C, networking, or dependencies.

The primary grammar snapshot covers all parsed declaration text, source lines,
nullable versus empty actions/types/destructors, symbol indices/types/usage,
fallback/wildcard, associativity/precedence, content flags, constituent ordering,
per-LHS rule chains, original and action-first rule IDs, aliases, NEVER-REDUCE,
inferred rule precedence, and used defines. The instrumentation includes the
pristine source in a separate driver and stops before FIRST sets. `-E` and `-g`
comparisons use the uninstrumented C executable. See the
[fixture guide](../testdata/grammar/README.md) for the exact snapshot format.

SQLite `parse.y` is compared with empty feature flags (the baseline grammar),
with `-l`, with UPDATE_DELETE_LIMIT/ORDERED_SET_AGGREGATES/MATH_FUNCTIONS enabled,
with TRIGGER/WINDOWFUNC/VIRTUALTABLE/CTE omitted, and with a define subsequently
undefined. Smaller fixtures exercise every declaration below, code-block
lexing, aliases, empty productions/actions, inline and named multiterminals,
precedence, error-symbol syntax, and conditional branches. Malformed fixtures
exercise scanner errors, parser recovery, duplicate declarations, preprocessor
syntax/exclusion errors, missing input, empty input, and invalid options.
The 1000/1001 RHS-symbol boundary is also covered through the Go command API.

Repeated offline runs check determinism and isolation between invocations.
Unimplemented stages are FIRST/nullable sets, configurations/states, lookahead,
conflicts, tables/compression, alias rewriting, C/template emission, and runtime
validation. Parsing `%start_symbol`, `error`, hooks, and destructors does not
implement their automaton/emitter/runtime effects.

## Command-line options

The complete `main.options` table is at **L1753–1780**. The table inventories
upstream behavior; the following status groups limit the Go claim:

| Options | Go support and verification boundary |
| --- | --- |
| `-D`, `-U`, `-E`, `-g`, `-l` | Implemented; differential grammar/preprocessor/reprint coverage. `-l` is verified for stored declaration text, not emitted C. |
| `-x` | Version early exit implemented and differentially verified; does not identify the pin. |
| `-s` | Implemented and differentially verified in `-g` mode; grammar counts are real, automaton/table counts remain zero as in C's `-g -s`. |
| `-f`, `-I`, `-O`, `-W` | Ignored flag-form placeholders implemented and differentially verified with empty/nonempty suffixes. |
| `-b`, `-c`, `-d`, `-m`, `-p`, `-q`, `-r`, `-S`, `-T` | Accepted with no effect in grammar-only modes, just as in C; this is differentially verified. Their generation/report/template behavior is not implemented or claimed. Normal generation fails explicitly. |
| `--grammar-snapshot` | Project-only extension, not an upstream option. Stops after inferred precedence assignment. |

No grammar-stage byte differences are permitted. Unknown-option syntax and
upstream help formatting are preserved; generation defaults remain future work.

| Option | Pinned C behavior |
| --- | --- |
| `-b` | Report basis items only rather than full closures; does not alter state construction. |
| `-c` | Skip `CompressTables` (default-reduce and SHIFTREDUCE optimizations). Does not disable the later packed-table algorithm. |
| `-dDIR` | Set output directory. Without it, output names follow the input pathname, despite help saying default `.`. Directory creation is not performed. |
| `-DNAME` | Add preprocessor macro name. `-DNAME=value` truncates at `=`; values are not expanded or evaluated. Duplicates are not deduplicated. |
| `-E` | Print preprocessed input plus newline to stdout and stop before grammar parsing/table generation. |
| `-fTEXT` | Ignored compiler-option placeholder. |
| `-g` | Reprint parsed grammar without actions to stdout; skip machine and artifact generation. This is not an exact round-trip of declarations or original formatting. |
| `-ITEXT` | Ignored compiler include-option placeholder, not a grammar include path. |
| `-m` | Emit makeheaders-style `.c` interface sections/include; suppress Lemon's `.h` generation. Does not run makeheaders itself. |
| `-l` | Suppress generated `#line` directives. |
| `-OTEXT` | Ignored compiler optimization-option placeholder. |
| `-p` | Show actions dropped by precedence in the `.out` report; no change to conflict resolution. |
| `-q` | Suppress `.out`, not `.c`/`.h`, diagnostics, or explicitly requested stats. |
| `-r` | Skip the whole `ResortStates` call; see the initialization caveat below. |
| `-s` | Print parser statistics to stdout. |
| `-S` | Also generate `.sql` containing symbol/rule/RHS metadata, not packed action tables. |
| `-x` | Print `Lemon version 1.0` and exit successfully before requiring a grammar filename; does not identify the SQLite source revision. |
| `-TFILE` | Use exactly this parser template, or diagnose failure to find/open it. |
| `-UNAME` | Remove the first matching previously defined macro (swap last entry into its place); command-line order matters. |
| `-WTEXT` | Ignored compiler warning-option placeholder. |

Supporting implementations: `handle_D_option`, `handle_U_option`,
`handle_d_option`, `handle_T_option` (**L1606–1667**); main dispatch and exit
paths (**L1783–1913**); `PrintAction` (**L3477–3540**); `file_makename`
(**L3302–3332**); SQL emission in `ReportTable` (**L4454–4534**).

### Argument syntax is not a standard modern flag parser

`OptInit`, `handleflags`, `handleswitch`, `OptNArgs`, and `argindex`
(**L2036–2230**) define these details:

- Exactly one non-option argument is required, except for `-x`. Options can
  appear before or after the filename.
- String arguments are attached: use `-Tfile`, not `-T file`. The latter passes
  an empty template name and treats `file` as another positional argument.
- A leading `-` sets a boolean flag; `+` clears it. String callbacks receive
  the suffix for either sign. Labels use prefix matching, so extra trailing
  characters on flag options are not rejected as conventional flag bundles.
- A bare `label=value` is recognized by `handleswitch`; flags in this form are
  errors. This is distinct from `-DNAME=value`, which takes the flag path.
  Ignored placeholders are safely ignored on the flag path; their null
  callbacks are not guarded on the bare `label=value` callback path. Do not
  assume all such spellings are safe.
- Although positional-count helpers recognize `--`, `OptInit` still processes
  it as an undefined option and does not stop option processing after it.
  It is not a working conventional end-of-options escape in this pin.
- Unknown options print syntax diagnostics and valid options, then exit 1.
  Normal generation returns 1 for errors or unresolved conflicts; early grammar
  error exit passes `lem.errorcnt` to `exit`, not necessarily the integer 1.

## Grammar declarations

The complete keyword dispatch in `parseonetoken` is at **L2600–2690**.
Every declaration below is **implemented and snapshot-verified for parsing and
stored metadata**, principally by `features-snapshot` and SQLite cases. Effects
on generated C, conflict resolution, and runtime behavior remain **not
implemented**, even where the upstream description below mentions them.
Keywords are case-sensitive. Unknown keywords diagnose.

Most single-argument declarations consume one identifier, quoted argument, or
brace block, **without a terminating period**. Quotes/braces delimit opaque text;
the outer delimiters are removed. Repeated text generally appends rather than
replaces (`WAITING_FOR_DECL_ARG`, **L2745–2808**). List declarations explicitly
end in `.`. The notation below is descriptive, not a new grammar definition.

| Declaration | Pinned meaning and argument shape |
| --- | --- |
| `%name NAME` | Generated API prefix; defaults to `Parse`. Template copying rewrites eligible `Parse` prefixes. |
| `%include {C}` | Accumulate C text near the start of generated source. Not a file-inclusion mechanism. |
| `%code {C}` | Accumulate trailing C text. |
| `%token_destructor {C}` | Destructor for terminal values; `$$` substitutes the value expression. |
| `%default_destructor {C}` | Default nonterminal destructor where no per-symbol destructor exists. |
| `%token_prefix PREFIX` | Prefix token constants in generated source/header. |
| `%syntax_error {C}` | Syntax-error hook; template exposes token value as `TOKEN`. |
| `%parse_accept {C}` | Acceptance hook. |
| `%parse_failure {C}` | Parse-failure hook, subject to runtime error-recovery compilation paths. |
| `%stack_overflow {C}` | Stack-overflow hook. |
| `%extra_argument {DECL}` | Extra parse-call argument, stored/fetched by generated `ARG_*` macros. |
| `%extra_context {DECL}` | Constructor/initialization context, stored/fetched by generated `CTX_*` macros. |
| `%token_type {TYPE}` | Terminal semantic C type; default `void*`. |
| `%default_type {TYPE}` | Default nonterminal semantic C type; absent this, untyped nonterminals use slot zero. |
| `%realloc FUNCTION` | Stack reallocator macro; default C `realloc`. |
| `%free FUNCTION` | Stack deallocator macro; default C `free`. Both explicit allocator declarations set `YYDYNSTACK=1`. |
| `%stack_size SIZE` | Emit stack-depth text; default 100, overridable by an existing `YYSTACKDEPTH`. Nonpositive depth enables growable stack; explicit allocator pair also enables growth. |
| `%start_symbol nonterminal` | Select start LHS instead of original first rule's LHS. Start symbol on any RHS is diagnosed. |
| `%left TOKEN ... .` | Assign one new precedence level and left associativity. Later declarations have higher precedence. |
| `%right TOKEN ... .` | Assign one new precedence level and right associativity. |
| `%nonassoc TOKEN ... .` | Assign one new precedence level; equal-precedence shift/reduce becomes ERROR. |
| `%destructor symbol {C}` | Per-symbol destructor text and source location. Emission uses specific destructors for nonterminals; terminals use `%token_destructor`. |
| `%type symbol {TYPE}` | Per-symbol datatype; duplicate explicit type diagnoses. Type slots are used for nonterminals, not arbitrary separate terminal types. |
| `%fallback BASE TOKEN ... .` | Assign BASE as fallback for each following token; first argument is the destination, not a source. Duplicate assignments diagnose. |
| `%token TOKEN ... .` | Predeclare terminals to control first-seen numeric order; declaration before use is not required. |
| `%wildcard TOKEN .` | Set one wildcard terminal; additional wildcard names diagnose. |
| `%token_class name TOKEN ... .` | New lowercase named multiterminal; its name must not already exist. Members are uppercase tokens, optionally prefixed by adjacent `\|` or `/`. |

Specialized validation is in `WAITING_FOR_DESTRUCTOR_SYMBOL` and
`WAITING_FOR_DATATYPE_SYMBOL` (**L2691–2727**), precedence parsing
(**L2728–2747**), and fallback/token/wildcard/class parsing (**L2809–2901**).
Destruction, datatype, allocator, and context emission is in
`emit_destructor_code`, `print_stack_union`, and `ReportTable`
(**L3853–3902**, **L4216–4322**, **L4565–4631**, **L4980–5020**).

### Other grammar syntax that affects compatibility

`parseonetoken` and `Parse` (**L2373–2599**, **L3138–3232**) also recognize:

| Construct | Pinned behavior |
| --- | --- |
| `lhs ::= RHS .` | Production; lowercase-initial LHS, uppercase tokens, lowercase nonterminals. Empty RHS allowed. Default RHS limit 1000. |
| `symbol(ALIAS)` | LHS/RHS semantic alias. Unused or duplicate aliases can diagnose later during emission, not initial grammar parsing. |
| `TOKEN\|OTHER` or `TOKEN/OTHER` | One compound RHS multiterminal. The separator must be adjacent to the following identifier to be scanned as one token. |
| Post-rule `{C}` | Reduction action; only one ordinary code fragment per rule. |
| Post-rule `[TOKEN]` | Explicit rule precedence; otherwise the first precedence-bearing RHS symbol/constituent supplies it, not the last. |
| Post-rule `{NEVER-REDUCE}` | Exact special marker setting `neverReduce`, not ordinary C action text. It does not itself add reduction code or remove a production. |
| `@ALIAS` in action text | Substitute RHS major token code rather than semantic value. |
| `/*LHS-overwrites-RHS*/` | Special exact action comment allowing direct overwrite of the leftmost RHS value; later uses of that RHS label diagnose. |
| `$$` in destructor text | Substitute the relevant semantic union field. |
| `error` | Reserved error-recovery nonterminal when present/used; not the uppercase terminal `ERROR`. |

Alias rewriting is textual, not a full C lexer (`translate_code`,
**L3968–4162**). C and C++ style comments are accepted by the grammar scanner;
identifiers and character classification are byte-oriented. Lack of an EOF
state check means the source does not promise comprehensive diagnostics for
unfinished grammar constructs. Representative malformed-input diagnostics and
acceptance of an incomplete final rule are differentially verified; exhaustive
malformed-input/crash equivalence is not claimed. All syntax above is stored by
the Go parser, but `@ALIAS`, overwrite comments, `$$`, and `error` have no
rewriting or runtime effects until later milestones.

## Grammar preprocessor directives

`preprocess_input` (**L3016–3072**) runs before scanning the grammar, comments,
or embedded C. All directives below are **implemented and differentially
verified**, including nested exclusion, defines/undefines, short-circuiting,
preserved newlines, and representative malformed-input diagnostics.

| Directive | Recognition and behavior |
| --- | --- |
| `%if EXPR` | At column one, with a literal space after `if`; include when evaluator returns true. |
| `%ifdef EXPR` | Same evaluation as `%if`; not limited to a single macro name. Literal space required. |
| `%ifndef EXPR` | Negate evaluated expression; literal space required. |
| `%else` | At column one with following whitespace; toggle inclusion at the current excluded depth. Trailing text is blanked, not parsed as an expression. |
| `%endif` | At column one with following whitespace; end excluded depth when applicable. Trailing annotations are allowed because the whole line is blanked. |

`eval_preprocessor_boolean` (**L2915–3009**) supports named defines, `!`, `&&`,
`||`, and parenthesized subexpressions; names begin with alphabetic bytes and
continue with alphanumeric bytes or `_`. Values after `-DNAME=` do not matter.
There is no `%define`, `%undef`, `%elif`, arithmetic, substitution, or C
`defined(...)` syntax. Macro usage is tracked during actual evaluation and
listed in the generated C prologue, so short-circuiting affects that metadata.

Do not substitute a conventional C-expression parser without comparison:
evaluation short-circuits the remainder of the current expression, not just
an operator's operand; for example the source can return true immediately at
`A || ...` if A is defined. Parentheses can constrain this. Skipped suffixes
and excluded nested conditionals need not be syntax-checked. Conditional
nesting tracks excluded regions rather than a general active-branch stack;
unmatched active `%endif` or an unterminated included branch is not uniformly
rejected. Excluded/directive bytes become spaces except for preserved newlines.
Indenting directives prevents recognition; column-one directive-looking text
inside a C block/comment is still processed in the initial pass.

SQLite examples: conditional grammar at `parse.y:L169–174`, fallback lists and
conditionals at `L270–291`, wildcard at `L294`, precedence at `L307–319`, and
named classes at `L324–332`. Ordinary `#ifdef` text inside `%include` is C
preprocessing for the generated file, not Lemon conditional syntax.

## Baseline pitfalls and verification boundary

These are source-derived cautions, **not executed test results**:

1. Pin both generator and template. Default template search includes a
   grammar-adjacent `.lt`, current-directory `lempar.c`, and executable/PATH
   lookup (`tplt_open`, **L3763–3816**). Record invocation directory, paths,
   explicit `-T`, define list/order, output filenames, compiler, and assertions.
   `-x` alone cannot verify the reference revision.
2. `-r` bypasses initialization work as well as sorting: `nxstate`, action
   counts, offsets, and default rule numbers are established only by
   `ResortStates` (**L1876–1879**, **L5293–5330**). Isolate this option in a
   baseline fixture; do not silently repair its behavior and call it matched.
3. `-S` emits grammar metadata only and changes the shared current `outname`
   when `.sql` is opened; later `#line` restoration uses that field
   (**L3337–3356**, **L4454–4534**, **L3831–3847**). Compare its generated C
   separately. `-c` still packs tables; `-q` only hides the report.
4. Original rule IDs, action-first emitted IDs, discovery state IDs, and
   renumbered state IDs differ. Token insertion order and `%token` affect
   constants. Never normalize away order changes before understanding them.
5. Generated files can exist on failure. Check process exit status, stdout,
   stderr, artifact contents, and expected artifact absence; distinguish
   preprocessing/reprint modes from full generation. Existing `.h` files may
   intentionally remain untouched when identical (`ReportHeader`, **L5111–5143**).
6. A generated SQLite `parse.c` is not independently linkable: its grammar
   actions require SQLite headers, functions, types, and runtime compile flags
   (`parse.y:L24–105`). Generic Lemon generation, compilation of a small parser,
   and integration with SQLite are different future verification layers.
7. Runtime macros in C `%include` text are not additional Lemon directives.
   SQLite sets `YYNOERRORRECOVERY`, `YYPARSEFREENEVERNULL`, and
   `YYMALLOCARGTYPE`, and conditionally its engine-on-stack macro
   (`parse.y:L72–105`). Runtime behavior must be compared with those settings
   explicit, not assumed from generic template defaults.

## Intentional deviations and exclusions

- Normal parser-generation requests fail with an explicit Milestone B message
  and no artifacts, rather than running an incomplete automaton/emitter.
- `--grammar-snapshot` is an inspection extension, not a generated parser file.
- Go uses per-invocation memory and immutable strings rather than process-static
  allocation pools and interned character pointers. Traversal and identity of
  named symbols remain deterministic; absent and allocated-empty text differ.
- ASCII byte classification matches the fixed C locale of the oracle. Other C
  locales, allocator failures, C integer-size/platform differences, and files
  changing while read are outside this milestone's verification claim.
- Undefined C behavior is not a compatibility requirement. In particular the
  pinned scanner can read beyond NUL in unfinished literals/comments inside a
  code block, and bare ignored-placeholder callbacks (`f=...`, `I=...`, `O=...`,
  `W=...`) can call a null function pointer. Go bounds scanning at EOF and ignores
  those placeholders safely. Exact C crash behavior is excluded. A missing RHS
  close-parenthesis diagnostic with no LHS alias uses `(null)` like this Linux C
  runtime, but that C null-string formatting is not portable.

There are no unexplained byte differences in the executed grammar comparisons.
No generated C/header/report/SQL compatibility or SQLite runtime integration is
claimed for the Go command. Milestone C requires separate authorization.
