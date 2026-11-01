# Pinned Lemon compatibility — Milestone F

Maintainer evidence: [development index](development/README.md). For the concise
user-facing boundary, see [compatibility reference](reference/compatibility.md).

## Status and pin

**Grammar processing, parser automaton construction, table generation, and C
emission are implemented and reference-verified for the cases below. Milestone F
passes the final pinned-output acceptance criterion.** All four pinned SQLite
baseline artifacts match byte for byte, not just under `diff -bw`. Milestone A's
sources and goldens remain unchanged. Generated-parser execution and a matching
SQLite source-tree build/test campaign provide supplementary evidence; this is
not a claim of exhaustive Lemon or SQLite feature/platform compatibility.

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
Alias rewriting and C/template emission are covered separately by E below;
snapshot-only comparisons do not establish their emitter/runtime effects.

### Milestone C automaton comparisons

`scripts/test-automaton.sh` separately compares C-derived saved goldens and Go
against C through `FindActions`, before compression or `ResortStates`. It uses
equivalent isolated input paths and compares full stdout, stderr, status, and
output inventories, both byte for byte and with `diff -bw`, retaining failures.
`--reference-only` independently repeats C execution; `--update-goldens` is the
explicit C-only regeneration command. Ordinary Go tests consume these same
goldens without executing C.

The [automaton matrix](../testdata/automaton/cases.tsv) covers nullable recursion,
suffix FIRST through nullable symbols and multiterminals, merged LALR lookaheads
(including an LR(1)-but-not-LALR(1) grammar), all shift/reduce precedence branches,
resolved/unresolved reduce/reduce and shift/shift conflicts, ordered compounds,
fallback/wildcard/error symbols, start selection and diagnostics, undefined
nonterminals, unreducible rules, action-first rule order, preprocessing and
malformed input. SQLite is tested under the same six default/feature/line-number
configurations as the grammar snapshot matrix.

Every normal symbol's nullable/FIRST set, every rule's start/reducibility flag,
every state's sorted basis/closure and lookaheads, and every action including
retained conflict/resolution markers are compared without renumbering or
normalization. Default SQLite matches **323 symbols, 187 terminals, 409 rules,
845 discovery-order states, zero conflicts**. These are uncompressed automaton
counts, not the number of states retained by later table optimization.
Fallback/wildcard metadata and `error` transitions are covered; runtime fallback,
wildcard dispatch, and error recovery are covered separately by the F tests below.

### Milestone D table comparisons

`scripts/test-tables.sh` compares **48 cases** against C-generated goldens and
freshly built C: all 33 automaton cases, six additional small/SQLite `-c`/`-r`
combinations, and nine targeted cases. It uses the same isolated-path,
stdout/stderr/status/inventory, byte and `diff -bw` policy. C-only regeneration
and independent repeat modes are explicit; ordinary Go tests use saved goldens
offline. See the [table matrix](../testdata/tables/cases.tsv) and
[snapshot schema](../testdata/tables/README.md).

The comparison includes all states after renumbering (including removed tails),
optimized actions and chain-bypass annotations, generated numeric action codes,
default rules, post-optimization reducibility, action counts, offset sentinels
and bounds, raw packed entries/holes, and every value of the final action,
padded lookahead, shift-offset, reduce-offset, and default arrays. Targeted
fixtures distinguish first-encountered default-frequency ties from rule order,
wildcard default suppression, absent versus explicit-empty code, multi-step
chain bypass, and the special `error` SHIFTREDUCE encoding.

Default SQLite matches **845 total states, 583 retained states, 2,207 action
entries, 2,341 raw lookahead entries, and 2,394 padded lookahead entries**.
All five Go numeric arrays also match every extracted value of the pristine
C-generated Milestone A `parse.c` baseline (2,207 / 2,394 / 583 / 413 / 583
values respectively). This validates numeric contents, not C text generation.
With `-c`, all 845 states remain and the action table has 17,281 entries.
With `-r`, C and Go retain zero states and pack no entries: the pinned behavior
skips initialization, not just sorting. It is intentionally not repaired.
SQLite feature/omit/undefine configurations also match in full.

Additional disposable validation compared 24 seeded, reachable, acyclic grammars
under default, `-c`, `-r`, and `-c -r` (96 successful comparisons). Tokens/rules
were permuted, nullable alternatives varied, and absent/empty actions mixed;
all cases reached table construction and matched stdout, stderr, and status.
No differences are normalized or permitted. C type selection, byte-size
accounting and template processing are verified separately by E, not by the
numeric-table snapshots. Generated runtime verification remains limited.

### Milestones E/F complete artifact comparisons

`scripts/test-generation.sh` uses **uninstrumented pristine C Lemon** and the
pinned `lempar.c`. All **57 cases** compare stdout, stderr, exit status, complete
artifact inventories, exact bytes, and `diff -bw` in equivalent isolated paths.
The `sqlite-baseline` case uses the exact `./lemon -S parse.y` invocation and
compares C artifacts against the unchanged Milestone A goldens before comparing
Go. `parse.c`, `.h`, `.out`, and `.sql` all match exactly. No whitespace or path
normalization is permitted. `-S`'s `.sql` line-restoration paths are preserved.

The [matrix](../testdata/generation/cases.tsv) covers all generation flags,
compression/resort combinations, default and feature/omit/undefine SQLite
configurations, basis/precedence reports, quiet/makeheaders modes, successful
and failing output directories, explicit/current/adjacent/executable template
lookup and missing templates. Small fixtures cover alias use/diagnostics,
major codes, overwrite markers, temporary LHS values, all hook declarations,
datatype hashing, shared actions/destructors, fallback/wildcard/error metadata,
empty/missing actions, NEVER-REDUCE, unresolved conflicts and malformed inputs.
Generated files on error are compared, not discarded. Unchanged-header timestamp
preservation and stale-header replacement are also tested offline.

Offline Go tests compare inventories and SHA-256 hashes derived only from C;
they require neither a C compiler nor networking. This avoids duplicating large
SQLite outputs for every configuration. The explicit differential command
always rebuilds and compares actual full files and retains failures. C-only
regeneration independently repeats every case; `--reference-only` reproduces
the saved expectations without Go. See the [fixture guide](../testdata/generation/README.md).

The arithmetic runtime fixture compiles both generated C files and checks
`7 - 2 * 3` produces value 1, exactly one acceptance, and no syntax errors.
Milestone F extends the original 52 cases with five recovery/stack configurations
described below. Disposable Milestone E differential probes additionally matched
type-width transitions around 127/255 symbols, 80 distinct semantic types with collisions,
999-byte template chunks/name boundaries, backslash/nested input paths,
output-directory basenames, and a failing SQL open with partial output.
Full PATH-only template search and permission-denied distinctions remain
implemented but not differentially verified. Integer-width boundary choices
above the exercised ranges are directly translated but not reference-verified.

### Embedded default template (issue #1)

The Go binary embeds the unchanged pinned SQLite 3.50.4 `lempar.c`. External
selection keeps its existing precedence: explicit `-T`, grammar-adjacent `.lt`,
current-directory `lempar.c`, then executable-directory lookup (when argv[0]
contains `/`) or PATH search (otherwise). Only an absent final search candidate
uses the embedded template. Explicit `-T` failures and errors reading selected
external files remain errors; the fallback does not repair invalid templates.
Unlike C Lemon, Go can therefore generate without any installed template. This
is an intentional standalone-executable extension, not an external precedence
change. A hash test pins the embedded bytes to the reference SHA-256.

Offline tests exercise competing, distinct overrides at every lookup level,
including first-match PATH order and failing explicit overrides. Without external
templates, both argv[0] lookup forms produce all four SQLite baseline artifacts
and process results byte for byte. The standalone portion of
`scripts/test-compatibility.sh` now uses the embedded fallback for both repeated
binary runs; C reference and differential fixtures still use the pinned disk
template. PATH override precedence is covered by Go tests, not a new C
differential case; permission-denied distinctions remain outside that coverage.

### Milestone F final acceptance and runtime checks

`scripts/test-compatibility.sh` runs offline `go test -count=3 ./...`,
`go vet ./...`, a cgo-disabled build, the independent C baseline, and all **58 grammar,
33 automaton, 48 table, and 57 full-generation differential cases**. It generates
the exact `./lemon -S parse.y` baseline twice with `PATH=/nonexistent` and an
unusable `CC`, comparing every artifact and process result against the original
goldens and each other using strict bytes and `diff -bw`. `readelf -l` confirms
there is no ELF dynamic interpreter. No output normalization or permitted
formatting differences are needed. The executable uses only the Go standard
library; a C compiler is required for the explicit oracle checks, not execution.

Six generated-runtime configurations run both pristine-C and Go output, with
assertions enabled and with `-DNDEBUG`: arithmetic precedence; fallback/wildcard,
recovery/continuation, EOF failure and token/nonterminal destruction (compressed
and `-c`); 400 nested pairs on a dynamic stack (compressed and `-c`); and an
eight-entry fixed stack's overflow/destruction/reuse. Full runtime output must
match checked expectations. The [fixture guide](../testdata/generation/README.md)
explains why pristine C's `-c` recovery timing changes which reductions and
destructors run. C and Go agree in each configuration; this is not an output
difference being excused.

The optional `scripts/test-sqlite.sh ARCHIVE` verifies the official source archive
hash/check-in and all pinned inputs, reads the actual configured generation
flags, compares all C/Go artifacts and process results, and compiles Go-generated
output into SQLite's default source-tree build and upstream Tcl testfixture.
It runs the unmodified regression files listed in
[`upstream-tests.txt`](../testdata/sqlite/upstream-tests.txt), independently in
fresh working directories, requiring successful status and nonempty zero-error
summaries. The source archive is a local input; tests never download it. The
[integration guide](../testdata/sqlite/README.md) gives requirements, exact build
policy, retained failure evidence, and reproduction instructions. This is a
selected regression campaign, not all SQLite `releasetest` or feature combinations.

Executed on Linux x86_64 with Go 1.27.3, GCC 12.2.0, GNU Make 4.3 and Tcl 8.6.13:
the default SQLite 3.50.4 build and all **13 selected regression files / 2,534
tests passed with zero errors**. Configured generation added
`-DSQLITE_ENABLE_MATH_FUNCTIONS -DSQLITE_THREADSAFE=1` before `-S parse.y`; all
artifacts still matched the original baseline goldens. The production default
build and upstream testfixture's distinct compile flags are both retained in
the build logs; neither is evidence of every possible compile-time setting.

## Command-line options

The complete `main.options` table is at **L1753–1780**. The table inventories
upstream behavior; the following status groups limit the Go claim:

| Options | Go support and verification boundary |
| --- | --- |
| `-D`, `-U`, `-E`, `-g`, `-l` | Implemented; differential grammar/preprocessor/reprint coverage; defines, undefines and suppressed C line directives also verified in E. |
| `-x` | Version early exit implemented and differentially verified; does not identify the pin. |
| `-s` | Differentially verified in `-g` and ordinary generation, including exact C type-size accounting and raw (not padded) lookahead count. |
| `-f`, `-I`, `-O`, `-W` | Ignored flag-form placeholders implemented and differentially verified with empty/nonempty suffixes. |
| `-c`, `-r` | Differentially verified in table snapshots and generated artifacts; `-r` deliberately retains the pinned zero-state behavior, not a repaired parser. |
| `-b`, `-d`, `-m`, `-p`, `-q`, `-S`, `-T` | Implemented and differentially verified for full generation/report/template behavior and their lack of effect in grammar-only modes. |
| `--grammar-snapshot` | Project-only extension, not an upstream option. Stops after inferred precedence assignment. |
| `--automaton-snapshot` | Project-only extension. Stops after conflict resolution, prints the graph even on automaton errors, returns 1 for errors or unresolved conflicts. Ignores generation/report flags, including `-s`; `-E` and `-x` still exit early. Takes priority over `-g` and `--grammar-snapshot`. |
| `--table-snapshot` | Project-only extension. Continues through compression, renumbering, and packing; honors `-c`/`-r`, retains snapshots on automaton errors/conflicts with status 1. Like the automaton mode, ignores reporting flags including `-s`; `-E`/`-x` exit early. Takes priority over other snapshot modes and `-g`. |

No grammar-stage byte differences are permitted. Unknown-option syntax and
upstream help formatting are preserved; ordinary generation is implemented.

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
stored metadata**, principally by `features-snapshot` and SQLite cases.
Precedence, start selection, and token-class automaton effects are additionally
covered by Milestone C. Effects on generated C are implemented and covered by
the generation fixtures and SQLite variants. Runtime execution covers the F
campaign above; declaration presence is not exhaustive runtime verification.
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
the Go parser; E additionally verifies the emitted alias/overwrite/destructor
rewriting and error-symbol metadata.

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

These are source-derived cautions; the `-r` numeric behavior is now executed
and covered by Milestone D fixtures:

1. Pin both generator and template. Default template search includes a
   grammar-adjacent `.lt`, current-directory `lempar.c`, and executable/PATH
   lookup (`tplt_open`, **L3763–3816**). Record invocation directory, paths,
   explicit `-T`, define list/order, output filenames, compiler, and assertions.
   Go's final embedded fallback pins the default bytes but does not override
   external files. `-x` alone cannot verify the reference revision.
2. `-r` bypasses initialization work as well as sorting: `nxstate`, action
   counts, offsets, and default rule numbers are established only by
   `ResortStates` (**L1876–1879**, **L5293–5330**). Small and SQLite table
   fixtures verify zero retained states; Go does not silently repair this.
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
   (`parse.y:L24–105`). Generic Lemon generation is verified by artifact checks;
   F's explicit integration script compiles the parser in the pinned source tree.
7. Runtime macros in C `%include` text are not additional Lemon directives.
   SQLite sets `YYNOERRORRECOVERY`, `YYPARSEFREENEVERNULL`, and
   `YYMALLOCARGTYPE`, and conditionally its engine-on-stack macro
   (`parse.y:L72–105`). Runtime behavior must be compared with those settings
   explicit, not assumed from generic template defaults.

## Intentional deviations and exclusions

- `--grammar-snapshot` is an inspection extension, not a generated parser file.
- `--automaton-snapshot` is likewise an inspection extension, not `.out` or a
  generated parser. Stage errors preserve C diagnostic text (including extra
  newlines for unreducible rules) and the boolean post-automaton exit status.
- `--table-snapshot` is also an inspection extension, not generated C. Numeric
  arrays are prepared separately from C rendering; neither emission-time action
  rewriting nor its diagnostics run. Original `NoCode` determines compression.
- Config builders and graph objects are per invocation rather than process
  statics/allocation pools. Propagation links are slices; their visitation order
  differs but the fixed-point sets and discovery/action order match the matrix.
- C's `actioncmp` final tie uses pointer subtraction. Go uses reverse creation
  order, matching descending addresses within a C 100-action allocation block.
  Subtraction between distinct blocks is undefined C behavior; matching the
  winner of cross-block duplicate-shift ties is excluded from the portable
  compatibility claim. Ordinary ordered shift/reduce/reduce ties do not depend
  on this: rule index breaks reduce ties. Small overlapping-compound fixtures
  verify shift/shift behavior within a block.
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
- Invalid start grammars can trigger C assertions after the initial diagnostic
  (for example a left-recursive start introduces a SHIFT/ACCEPT pair not handled
  by `resolve_conflict`). Go retains the start diagnostic and failure status;
  reproducing C aborts is excluded. The RHS-start golden uses right recursion.

There are no unexplained byte differences in the executed grammar, automaton,
table, or complete artifact comparisons. Generated C/header/report/SQL
compatibility is claimed only for the documented matrix and recorded platform.
Milestone F's pinned-reference acceptance is complete. Cross-platform/locale
behavior, exhaustive template/filesystem/large-integer boundaries, allocator
failures, every feature combination, and the full SQLite regression suite remain
outside the executed verification claim. These are limits, not known Go/C output
differences; the undefined-C and project-extension exclusions above still apply.

## Separate generated-Go contract — issue #13

`glcc` and `glemon` share a Go emitter/runtime, not the C output ABI or CLI.
SQLite's C-action `parse.y` remains a C-only acceptance fixture. No Go output
is compared textually with SQLite's C output. The [Go reference](reference/go-parser.md)
specifies supported declarations and intentional differences: ordinary Go flags,
package selection, Go action/type syntax, zero-valued actionless reductions,
Go-style extra parameters, dynamic default stack, and Go template overrides.
C allocation directives are rejected rather than ignored.

`TestGoParsers` executes isolated generated modules with and without compression:
precedence/associativity/parentheses, heterogeneous values, empty productions,
fallback/wildcard, recovery and error suppression, failure/accept hooks,
destructors/partial cleanup, deep dynamic stacks, fixed overflow (including
empty reductions), parameter/alias naming, reuse, and no-action grammars.
Generation is repeated through both names and checked for identical gofmt output.
Command build/invocation, malformed grammar/options, preprocessing, explicit
template override, and the tutorial's actual Go example are also tested offline.

`scripts/test-go-runtime.sh` separately compiles the pristine C reference and
compares executable Go/C results for `testdata/go/runtime.y`, with target-specific
parameter declarations and drivers. Both commands and compression settings are
covered. The recovery probe also confirmed that C itself can differ in reduction
and recovery timing with compression disabled; that difference is not repaired
in the shared frontend. These checks establish the tested Go contract, not
exhaustive equivalence for all grammars, actions, templates, or malformed streams.

## LL(1) Go target — issue #14

`glime` is an independent LL(1) generator, not a compatibility mode for the
LALR runtimes or a reconstruction of historical Lime. It shares only the
parse-only Lemon frontend. Supported declarations, complete-input API, typed
zero-value semantics, action timing, hooks and explicit exclusions are specified
in the [glime reference](reference/glime.md). The recursive strategy is reserved
and explicitly unimplemented. No automatic grammar transformation is performed.

Offline `internal/ll1` tests verify FIRST/FOLLOW/SELECT and nullable propagation,
EOF conflicts, all competing rule pairs, left recursion through nullable prefixes,
nonproductive/undefined symbols, unreachable rules, deterministic explanation,
and frontend immutability. Executable generated-module cases cover the actual
calculator grammar (precedence and left-associative subtraction), heterogeneous
values, nested completion order and failure side effects, epsilon/actionless
rules, deep nesting, invalid token kinds and malformed streams. This is not a
claim of equivalence to SQLite's C actions or support for every Lemon directive.

## Shared PL/0 application example — issue #15

The [example contract](reference/pl0.md) specifies one PL/0 dialect implemented
by `glcc` and `glime` Go frontends with one shared AST, lexer, name checker, and
interpreter. Cross-frontend AST equality includes source positions; independent
expected outputs and expression shapes verify precedence, left associativity,
lexical scope, nested/direct/mutual recursion, empty statements, and integer
semantics. Malformed sources, static errors before output, runtime division
errors, writer failures, CLI behavior, and exact parser regeneration are tested.

This is application-level Go coverage, not a historical PL/0 compiler replica
or an extension of SQLite C compatibility. `write`, comments, forward-visible
procedures, zero initialization, and Go int64 arithmetic are documented example
choices. The pinned sources, fixtures, and all generator code remain unchanged.
