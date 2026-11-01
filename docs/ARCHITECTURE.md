# Pinned Lemon architecture — Milestone F

Maintainer record: [development index](development/README.md). For a conceptual
introduction, see [how Lemon works](explanation/how-lemon-works.md).

## Scope and evidence

This document describes the complete C pipeline and the implemented Go grammar
front end, parser automaton, numeric tables, and C emitter. Milestone E adds
template processing, semantic-action rewriting, and C/header/report/SQL output.
The generated runtime comes from the pinned original C template, not a Go
parser engine. Embedded C is rewritten textually, not parsed as C.

Milestone F adds validation, not a new parsing algorithm or runtime layer.
`scripts/test-compatibility.sh` joins the offline tests, independent C baseline,
intermediate differential snapshots, and complete artifact comparisons. It also
checks that the cgo-disabled ELF executable has no dynamic interpreter and can
generate the exact SQLite baseline twice with an unusable PATH and CC.
Generated-runtime fixtures run both C and Go output, with and without assertions,
and include recovery/destruction and growable/fixed stacks.
`scripts/test-sqlite.sh` accepts a local hash-pinned archive, compares configured
C/Go parser artifacts, then installs only Go's executable/output into a fresh
SQLite build directory and compiles/runs selected upstream tests. This optional
source-tree build never changes the vendored reference or runs in ordinary Go
tests. See [the integration guide](../testdata/sqlite/README.md).

The reference is SQLite **3.50.4**, Fossil revision
`4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3`.
Line references below refer to the checked-in reference snapshot, not a moving
upstream branch. All 6,056 lines of
[`lemon.c`](../reference/sqlite/tool/lemon.c) were read for this document.
`lemon.c:Lx–Ly` means inclusive lines in that file; companion references are
[`lempar.c`](../reference/sqlite/tool/lempar.c) and
[`parse.y`](../reference/sqlite/src/parse.y). See
[REFERENCE.md](REFERENCE.md) for provenance and executed baseline checks;
source analysis alone does not establish Go executable compatibility. The
Milestone B snapshot and command comparisons are documented in
[COMPATIBILITY.md](COMPATIBILITY.md).

### Implemented Go ownership and translation boundaries

`cmd/lemon/main.go` passes process arguments and streams to `lemon.Run`, then
exits with its status. Everything else lives in `internal/lemon`:

| File | C correspondence and responsibility |
| --- | --- |
| `lemon.go`, `options.go` | Grammar-stage `main`, option scanning, D/U callbacks, file input, early exit modes. |
| `preprocess.go` | `eval_preprocessor_boolean`, `preprocess_input`; byte offsets and used-define tracking. |
| `scanner.go` | The scanning portion of `Parse`; comments, literals, code blocks, token source lines. |
| `parser.go` | `pstate`, `parseonetoken`; same parser states, recovery boundaries, and declaration slots. |
| `grammar.go` | Grammar, symbol, and rule ownership; `Symbol_new`/lookup, `Symbolcmpp`, action-first numbering/`Rule_sort`, `FindRulePrecedences`. |
| `output.go` | `Reprint`/`rule_print` and project-only deterministic snapshot serialization. |
| `automaton.go` | Byte sets, `FindFirstSets`, configuration builder/closure, `FindStates`/`getstate`/`buildshifts`, `same_symbol`, `FindLinks`, `FindFollowSets`. |
| `action.go` | `Action_add`/`Action_sort`, `FindActions`, `resolve_conflict`, original action enum values and retained loser actions. |
| `table.go` | `CompressTables`, `ResortStates`/`stateResortCompare`, `compute_action`, `acttab` allocation/action/insertion/sizing, and `ReportTable` numeric packing/array preparation and table snapshots. |
| `template.go` | Output naming/opening, template search, 999-byte chunk transfers and parser-name substitution, logical line counting, C integer type selection. |
| `code.go` | `translate_code` alias/major-code/overwrite rewriting, destructor-call prefixes/suffixes; `print_stack_union` datatype hashing and semantic union. |
| `emit.go` | `ReportTable` orchestration, control macros, numeric array rendering and byte accounting, destructor/reduction switch sharing, hooks and line restoration. |
| `report.go` | `ReportOutput`, `RulePrint`/`PrintAction`, `ReportHeader` (unchanged-file preservation), and SQL metadata. |

All mutable tables, define usage, parser state, and diagnostics belong to one
invocation; unlike C's process statics, repeated Go calls do not share state.
Named symbol identity is represented by pointers. Lookup maps never supply
iteration order: insertion-order slices supply symbol order, stable partitioning
preserves it within symbol groups, and unique `IRule` values sort rules. Global
rule order is a slice, while the distinct reverse-source-order `NextLHS` chain
is retained. Anonymous compounds have independent symbol objects; references
to named token classes share the same mutable symbol, just as in C.

Go strings own immutable byte text rather than interned C buffers. Content
equality replaces interned-string pointer equality; nullable `*string` fields
retain the crucial absent/explicit-empty distinction. Future emission must
continue comparing text by value, not by Go string-pointer address. C allocation
pools and hash bucket layout are not emulated because they do not determine
grammar-stage traversal. All classification uses explicit ASCII byte predicates
matching the oracle's `LC_ALL=C`, not Unicode or ambient locale rules.

`--grammar-snapshot` stops after `FindRulePrecedences`; upstream `-g` stops
before that inference and therefore displays explicit precedence marks only.
The C instrumentation driver includes pristine `lemon.c` and repeats only its
grammar-stage initialization/indexing, stopping at the same snapshot boundary.
It never substitutes for the uninstrumented final-artifact oracle. Snapshot
schema and commands are in [the fixture guide](../testdata/grammar/README.md).

`--automaton-snapshot` runs through `FindActions` but never calls
`CompressTables` or `ResortStates`. Its separate C driver includes the same
pristine source and stops at that boundary. The [automaton fixture guide](../testdata/automaton/README.md)
defines the representation. Go config builders are local to each recursive
`getstate` call instead of shared process statics. Structural basis keys encode
ordered `(original rule index, dot)` pairs; maps only look up identities.
Propagation links use slices rather than pool-allocated singly linked lists;
link visitation order does not affect the fixed-point sets or state discovery.
Actions sort before resolution, not afterwards: mutated types can intentionally
appear out of enum order. Equal-symbol/type shift ties use reverse creation
order rather than Go heap addresses. C's final pointer tie-break is only
portable within an allocation block; cross-block ties are not a portable C
contract (see compatibility exclusions).

`--table-snapshot` continues through `CompressTables` (unless `-c`),
`ResortStates` (unless `-r`), and `BuildTables`. `Tables` owns the five numeric
arrays and retains raw packed entries and offset bounds for inspection. State
objects retain identity while `States` is reordered, so incoming SHIFT targets
automatically see renumbered IDs. Rule `Index` remains the original identity;
numeric reductions use `IRule`. The table driver copies the packing block from
`ReportTable` without running template processing or code translation. No C
types, table byte-size accounting, fallback-array emission, or C text rendering
are implemented in D's snapshot mode. E's ordinary generation path adds these
without changing the snapshot contract. See [table snapshots](../testdata/tables/README.md).

Ordinary generation now follows the full pinned `main` flow below. `cOutput`
counts generated newlines while preserving C's historical uncounted `#ifndef`
and skipped-template-header line quirks. Semantic union assignment runs before
action rewriting. Equal interned C strings are compared by value in Go,
including nil-versus-empty distinctions, so identical reduction code and
destructors share cases without depending on Go string addresses. Report output
runs before translation mutates rules/destructor source lines, as in C.
`BuildTables` remains the numeric source of truth; rendering adds no separate
packing algorithm. See [generation fixtures](../testdata/generation/README.md).

## 1. Main execution flow

`main` (`lemon.c:L1739–1913`) runs a single grammar per process:

1. `OptInit` processes options; `-x` exits before requiring a filename. Otherwise
   exactly one positional filename is required. Defines are sorted, the string,
   symbol, and state tables initialized, and `$` inserted first.
2. `Parse` reads and preprocesses the grammar, scans tokens, and calls
   `parseonetoken`. `-E` exits after printing preprocessed input. Grammar errors
   also exit here; an empty grammar is rejected.
3. Find `error`, add `{default}`, partition/index symbols, and assign generated
   rule numbers: rules with explicit action text first, then rules without.
   Save the original first rule in `startRule` before `Rule_sort` changes the
   global rule list (`L1790–1838`).
4. `-g` calls `Reprint` instead of building a machine. Normal generation calls
   `SetSize`, `FindRulePrecedences`, `FindFirstSets`, `FindStates`,
   `State_arrayof`, `FindLinks`, `FindFollowSets`, and `FindActions`, in order.
5. Unless `-c`, call `CompressTables`; unless `-r`, call `ResortStates`.
   Write `.out` unless `-q`; call `ReportTable` for `.c` and optionally `.sql`;
   write `.h` with `ReportHeader` unless `-m` (`L1840–1890`).
6. Optionally print statistics; report conflict count; return failure if there
   are errors or unresolved conflicts. Normal completion calls
   `lemon_free_all`; some early paths call `exit` directly (`L1892–1913`).

Generation is not transactional: diagnostics can coexist with generated files,
and some errors occur during emission. File existence alone proves neither
successful generation nor a usable parser.

## 2. Grammar preprocessing, scanning, and parsing

`Parse` (`L3079–3232`) opens the input in binary mode, measures it, reads the
whole file into a NUL-terminated mutable buffer, and rejects sizes above
100,000,000 bytes. `preprocess_input` runs **before** lexical analysis, including
before recognition of code blocks or comments. It blanks directive lines and
excluded text with spaces while preserving newlines (`L3016–3072`).

Conditional directives must begin at column one. `%if `, `%ifdef `, and
`%ifndef ` require the literal space in their recognized prefixes; `%else`
and `%endif` require following whitespace. `eval_preprocessor_boolean`
(`L2915–3009`) tests membership in the command-line define list and handles
`!`, `&&`, `||`, and parentheses. This is not a C preprocessor: there is no
macro substitution, numeric expression language, `defined()`, or `%elif`.
The evaluator short-circuits by returning from the current expression; do not
assume conventional C operator precedence or complete validation of skipped
suffixes. The preprocessor tracks excluded depth rather than a full stack of
all active conditionals, so malformed nesting is not comprehensively checked.
`-E` prints the resulting buffer with an additional newline and does not parse
the grammar (`L3129–3135`).

The scanner skips whitespace and C/C++ comments, recognizes double-quoted
arguments, nested-brace C blocks (ignoring braces inside comments and quoted
literals), alphanumeric/underscore identifiers, `::=`, adjacent `/NAME` or
`|NAME` tokens, and single-character punctuation (`L3138–3241`). Top-level
quoted arguments do not use the code-block scanner's escape handling.
Tokens are temporarily terminated in-place; closing quote/brace is excluded
while the opening delimiter remains. `Strsafe` interns each token before the
buffer is restored. Input is byte-oriented C text, not a Unicode grammar.

`parseonetoken` is a hand-written state machine, not a generated parser
(`L2373–2910`; `pstate` and states at `L2321–2370`). It builds declarations
and `lhs(alias) ::= rhs(alias) ... .` productions; empty RHSs are legal.
Lowercase initial letters select rule LHS names, uppercase initial letters
select tokens. A post-rule `{...}` attaches action text; `[TOKEN]` overrides
rule precedence. `{NEVER-REDUCE}` is a special marker, not ordinary action
text. Adjacent alternatives and `%token_class` form multiterminals. Rule or
declaration errors resynchronize at `.` or `%`. There is no final parser-state
validation at EOF in `Parse`; do not infer rejection of every incomplete input.
Default `MAXRHS` is 1000 (5 with `TEST`; `L45–49`).

Declaration arguments are accumulated in designated string slots, sometimes
with inserted `#line` directives (`L2745–2808`). They are opaque C text until
the emitter's limited rewriting; there is no C compiler frontend here. See
[COMPATIBILITY.md](COMPATIBILITY.md) for the complete directive inventory.

## 3. Symbols, rules, actions, and associative tables

Principal structures are defined at `L314–505`:

| Structure | Meaning and important identities |
| --- | --- |
| `symbol` | Interned name, kind, assigned index, productions, fallback, precedence/associativity, FIRST/nullable, destructor/type metadata, semantic-content flag, optional ordered constituent list. |
| `rule` | LHS/RHS and aliases, source locations, original `index`, generated `iRule`, code/prefix/suffix, precedence symbol, reduction flags; `next` and `nextlhs` are different lists. |
| `config` | `(rule, dot)` LR item, per-item lookahead set `fws`, forward/back propagation links, owning state, algorithm status; `next` is closure order, `bp` basis order. |
| `state` | Sorted basis and full closure, number, action list, action counts, packed offsets, default-reduction and auto-reduction metadata. |
| `action` | Lookahead symbol, action kind, state-or-rule payload, optimization annotation; conflict/resolution/deletion kinds remain in the list. |
| `lemon` | Per-run graph, counts, emission settings/text, filenames, errors, and generated numeric action boundaries. Not all actual state lives here. |

`Strsafe` deduplicates text (`L5409–5540`). `Symbol_new` interns symbols and
increments `useCnt` even for existing symbols; uppercase names become terminals
and other names nonterminals (`L5543–5567`). `$` is the special index-zero EOF
symbol despite not having an uppercase name. `{default}` is a pseudo-symbol
outside the normal `nsymbol` range after sorting. Named multiterminals sort
after it and are excluded from normal symbol counts (`L1804–1817`). Anonymous
RHS multiterminals are separate objects (`L2500–2530`).

Rules are appended in source order globally, but prepended to their LHS's
`nextlhs` chain (`L2450–2490`). Keep both orders: they influence state
construction and output differently. `index` remains original identity even
after `iRule` and the global list are reordered.

The generated associative-array code implements four process-static tables:
interned strings (`x1a`, `L5424–5540`), named symbols (`x2a`, `L5597–5743`),
states (`x3a`, `L5790–5931`), and current-builder configurations (`x4a`,
`L5941–6056`). Hash chains serve lookup, but contiguous insertion-order records
serve `Symbol_arrayof` and `State_arrayof`. State keys compare the sorted basis
sequence of original rule indices and dot positions, **not lookahead sets**
(`Configcmp`, `statecmp`, `statehash`, `L5746–5780`). Configuration keys are
the same pair, scoped to the builder, which is cleared on reset.

## 4. Precedence, nullable, and FIRST

`FindRulePrecedences` (`L889–911`) retains explicit `[TOKEN]`, otherwise chooses
the **first** RHS symbol with precedence, scanning multiterminal constituents
in order. This is not yacc's customary rightmost-terminal rule.

`FindFirstSets` (`L918–973`) first clears nullable flags and allocates
nonterminal FIRST sets. A fixed point marks a LHS nullable when every RHS symbol
is nullable (including an empty RHS). A second fixed point unions FIRST sets
along nullable prefixes; a terminal adds itself and stops, a multiterminal
adds all constituents and stops. Self-recursion is handled explicitly.
`SetSize`, `SetNew`, `SetAdd`, and `SetUnion` (`L5336–5385`) use byte arrays,
not packed bitsets; mutation returns whether anything changed. Preserve that
change/no-change contract if the Go representation eventually differs.

## 5. Configurations and LR(0) state construction

`FindStates` (`L981–1036`) chooses `%start_symbol` or the original first rule's
LHS; a missing specified name diagnoses and falls back. It diagnoses any RHS
occurrence of that start symbol instead of introducing an augmented start
production. All productions for the start LHS enter the initial basis at dot
zero with EOF in `fws`, and get `lhsStart` set.

`Configlist_add` / `Configlist_addbasis` deduplicate `(rule.index, dot)` in the
current builder, allocate an empty lookahead set, and append to closure and/or
basis lists (`L1424–1476`). `Configlist_closure` (`L1479–1521`) expands a
nonterminal after the dot into dot-zero items for its productions. FIRST of the
remaining suffix seeds those items' `fws`; if the suffix is nullable, add a
forward propagation link from the parent item. An undefined nonterminal is an
error except for the reserved `error` symbol.

`getstate` (`L1043–1086`) sorts the basis and looks it up. An existing state
receives the candidate's backward propagation links, and candidate items are
discarded. A new state receives its closure, sorted item lists, and the next
discovery number; it is inserted **before** recursively building successors.
`buildshifts` (`L1108–1151`) visits sorted closure items, groups matching
symbols, advances dots into a successor basis, and adds backward links to
predecessor items. `same_symbol` (`L1092–1103`) compares multiterminals by their
ordered constituents; this is not unordered set equality. A multiterminal
transition emits a shift for each constituent token.

## 6. Lookahead propagation and conflicts

`FindLinks` (`L1156–1183`) assigns owning states and reverses backward links
into forward links. `FindFollowSets` (`L1191–1222`) repeatedly unions source
`fws` into linked destinations, revisiting changed items until convergence.
These are **configuration-specific LALR lookaheads**, not a single global
FOLLOW set for each grammar nonterminal.

`FindActions` (`L1229–1319`) adds a reduce for each completed item/lookahead,
adds ACCEPT on the start nonterminal in state zero, sorts actions, resolves
same-symbol conflicts, and diagnoses rules that have no surviving REDUCE.
`actioncmp` (`L593–609`) orders by symbol index, action enum, original rule
index for reduce-like actions, then pointer difference. SHIFT precedes REDUCE;
`resolve_conflict` relies on that (`L1318–1378`).

Conflict behavior is explicit, not “pick any valid parse”:

- Shift/shift keeps the first and marks the other `SSCONFLICT`.
- Shift/reduce without sufficient precedence keeps shift, marks the reduce
  `SRCONFLICT`, and increments conflicts. Higher precedence wins; equal right
  associativity chooses shift, left chooses reduce; equal nonassociativity
  changes the shift to ERROR.
- Reduce/reduce without distinguishing precedence keeps the earlier sorted
  reduce and marks the other `RRCONFLICT`. Distinct known precedence chooses
  the higher rule precedence.
- Resolved losers remain as `SH_RESOLVED` / `RD_RESOLVED`; `-p` controls their
  report visibility, not resolution. Conflict markers are not emitted as live
  table actions (`compute_action`, `L3690–3717`).

## 7. Compression, sorting, and renumbering

There are several distinct orderings; they are not interchangeable:

- `Symbolcmpp` groups EOF/tokens, nonterminals/default, then multiterminals,
  preserving initial insertion indices within groups (`L5570–5590`).
- `Rule_sort` uses `iRule`: explicit actions first, stable source order within
  the two assigned groups; `index` still keys configurations (`L1670–1715`,
  `L1819–1834`).
- `CompressTables` (`L5153–5259`) chooses the most frequent eligible REDUCE as
  a default, excluding start-LHS rules and states with a wildcard SHIFT.
  Ties retain the first encountered candidate. One action gets `{default}`;
  duplicates become `NOT_USED`. States with no remaining SHIFT or different
  REDUCE become `autoReduce`. Incoming shifts become SHIFTREDUCE; no-code,
  single-RHS nonterminal chains can be bypassed with `spOpt` annotations.
- `ResortStates` initializes action counts, offset sentinels, and default rule
  numbers, then sorts all but state zero by descending nonterminal count,
  terminal count, and old state number. It renumbers and trims trailing
  auto-reduce states from `nxstate` (`L5268–5330`). `nstate` remains the
  full count, important for action-code ranges.

**Pinned-source caveat:** `-r` skips the *whole* `ResortStates` call, not just
sorting. That also skips initialization of `nxstate`, action counts, offsets,
and default metadata. `lem` starts zeroed. It is not a sound “same parser,
original numbering” baseline (`L1792`, `L1879`, `L5293–5330`). Milestone D
fixtures verify this exact behavior: zero retained states, no packed entries,
empty action/offset/default arrays, and a terminal-count-sized padded lookahead
array. Go intentionally does not repair the pinned implementation.

## 8. Packed table construction and numeric encoding

`ReportTable` first sets numeric action ranges (`L4433–4438`): shifts use full
state numbers, SHIFTREDUCE begins at `nstate`, then error/accept/no-action,
then reductions at `minReduce + iRule`. `compute_action` converts nonterminal
SHIFTREDUCE to REDUCE except for `error`; it returns negative for discarded
actions (`L3690–3717`). During earlier `ResortStates`, these boundaries are
still zero, so `iDfltReduce` stores a rule number, not the final action code.

Terminal and nonterminal action sets are separate transactions. `axset_compare`
sorts them by descending size with original transaction order as tie-breaker
(`L4366–4388`, `L4634–4709`). `acttab_action` accumulates `(lookahead, action)`;
`acttab_insert` first searches backward for an exact reusable transaction,
then forward for collision-free holes, then writes entries and returns the
offset (`L648–873`). Rejecting accidental matches for tokens *absent* from a
transaction is as important as fitting its present entries. Terminal placement
uses `makeItSafe=1` to guarantee safe lookahead indexing; nonterminals permit
negative offsets for tighter packing.

Emission writes `yy_action`, `yy_lookahead`, `yy_shift_ofst`, `yy_reduce_ofst`,
and `yy_default` (`L4770–4932`). A lookup adds symbol code to the state offset,
checks the lookahead tag, then uses the action or default. Action storage trims
unused trailing entries; lookahead storage includes safety padding and is not
necessarily the same length. `NO_OFFSET` is an internal negative sentinel;
emitted terminal and nonterminal missing offsets use different replacements.
`minimum_size_type` has exact boundary tests, including strict `<65535` and
`<32767`, and assumes 1/2/4-byte C types (`L4337–4361`). Reported `tablesize`
is an accounting estimate, not the size of generated C or every runtime table.

## 9. Template, code transformation, and artifacts

`file_makename` changes the input suffix; absent `-d`, artifacts follow the
input pathname, **not necessarily the working directory**, despite option help
(`L3302–3356`). It uses `/` for output-directory basename extraction.
`tplt_open` checks explicit `-T`, then grammar-adjacent `.lt`, current-directory
`lempar.c`, then executable-directory/PATH lookup (`L3640–3684`,
`L3763–3816`). Pin template selection and invocation location for comparisons.
Go adds a final embedded fallback only when the external search finds no file.
`reference/template.go` embeds the pristine pinned `sqlite/tool/lempar.c` directly,
without a duplicate generated copy. `openTemplate` converts it to bytes for the
existing emission path; explicit `-T` and selected-file read failures still error.

`tplt_xfer` copies fixed-size line chunks until a column-one `%%` marker and
rewrites eligible `Parse` prefixes to `%name`; the separator sequence is a
positional protocol, not named substitution slots (`L3728–3749`).
`ReportTable` interleaves template sections with include text, token defines,
control macros and semantic union, packed tables, fallback/name/rule tables,
destructors, overflow hook, rule LHS and negative-RHS-length tables, reduce
cases, failure/error/accept hooks, and trailing `%code` (`L4412–5108`).
The template header can be skipped depending on the initial `%include` text.

`print_stack_union` assigns terminal type slot zero, nonterminal slots by a
specific unsigned hash and linear probing, and a special `error` slot
(`L4216–4322`). `translate_code` (`L3968–4162`) rewrites aliases to stack or
temporary semantic values, `@alias` to major token codes, and handles
`/*LHS-overwrites-RHS*/`. It diagnoses unused/duplicate aliases and generates
destructor prefixes/suffixes for unreferenced RHS values. Its textual scanning
is not C-aware enough to promise preservation inside comments or literals.
`emit_destructor_code` substitutes `$$` (`L3853–3902`); identical interned
reduce text/prefix/suffix can share switch cases (`L5033–5059`). Emission mutates
rules and destructor metadata and should not be treated as a pure repeatable
render of an immutable graph.

`ReportOutput` writes states, actions, FIRST/nullable data, syntax-only symbols,
and rules (`L3543–3634`). `ReportHeader` avoids rewriting an identical header
(`L5111–5143`). `-S` writes SQL **symbol/rule/RHS metadata**, not a dump of the
packed action/state tables (`L4454–4534`). Opening it also changes `outname`,
which subsequent generated `#line` paths use; compare `-S` output separately
from ordinary generation rather than assuming an otherwise identical `.c`.

## 10. Generated runtime boundary

`lempar.c` is the **C parser engine template**, not the grammar-file scanner.
`ParseInit` / `ParseAlloc` establish an instance; `Parse` consumes one supplied
token per call, with zero as EOF, reducing as needed before shifting or
accepting (`lempar.c:L344–381`, `L868–1069`). `yy_find_shift_action` tries exact
lookup, then fallback, then wildcard (not for EOF), then default; nonterminal
goto uses `yy_find_reduce_action` (`L538–628`). Fallback chains are constrained
by runtime assertions, not fully validated in grammar parsing.

`yy_reduce` executes inserted actions, uses rule LHS and negative RHS lengths,
and pushes the goto result; stack entries may hold pending reduce codes, not
only state numbers (`L731–778`). `yyGrowStack`, `ParseFinalize`, and `ParseFree`
manage instance storage and destructors (`L297–325`, `L441–485`). Error-symbol
recovery, no-recovery mode, and ordinary token-discard recovery are different
compile-time paths (`L965–1049`). These C runtime semantics come from the
original template. E's arithmetic smoke test executes generated C, but
exhaustive runtime equivalence and SQLite integration remain outside the
current verification claim.

SQLite's `parse.y` demonstrates this boundary: C types/context/hooks and custom
stack allocation (`parse.y:L24–57`), parser renaming and `YYNOERRORRECOVERY`
(`L59–80`), original start rule and conditional NEVER-REDUCE marker
(`L163–174`), alias/overwrite/major-code syntax (`L183–190`), token ordering,
fallback, wildcard, precedence and token classes (`L253–332`). Compiling its
generated C requires SQLite's surrounding headers and implementation; a
standalone Lemon compile is not a standalone SQLite parser build.

## 11. Global state and C-to-Go hazards

Despite the historical “uses no global variables” comment above `struct lemon`,
the implementation has substantial process-static mutable state: allocation
list (`L65–128`), action pool (`L569–585`), configuration builder and free list
(`L1385–1421`), defines/options/output/template settings (`L1598–1667`,
`L1739–1784`, `L2036–2038`), propagation-link pool (`L3252–3302`), string
builder (`L3930–3960`), set width (`L5336–5341`), and all four associative tables.
Initialization is often “if already initialized, return.” Cleanup does not
reset all static pointers. This is a one-shot executable, not a reentrant
library; repeated or concurrent Go invocations must not accidentally share it.

Important fidelity hazards for subsequent milestones:

- Preserve discovery, insertion, list, tie-break, and output order explicitly;
  Go map iteration is not a replacement for the table arrays. Pointer-based
  action ties need deliberate treatment, not Go address ordering.
- Preserve object identity and interning semantics: C compares symbols and
  deduplicated action strings by pointer, while state keys are structural.
  Distinguish absent action from an explicit empty action and from
  `{NEVER-REDUCE}`; these influence numbering and optimization.
- Keep original rule IDs separate from emitted rule IDs, and full state counts
  separate from retained state counts. Do not cache emitted action codes before
  their boundaries are initialized.
- Handle tagged state/rule action payloads and all conflict marker kinds
  explicitly rather than reproducing C unions or assuming every action is live.
- C character classification casts to unsigned bytes (`L16–21`); locale and
  byte behavior differ from Go Unicode classification. Hash arithmetic uses
  unsigned wrapping; datatype hash layout affects generated union field names.
- `lemon_free` only zeroes allocations; actual release is deferred to
  `lemon_free_all`, and `lemon_realloc` can retain storage (`L81–133`). Do not
  blindly imitate this ownership model with Go slices sharing mutable backing
  arrays. Token buffers, interned text, and graph references have different lives.
- Diagnostics, exit status, stdout/stderr, whitespace, filenames, `#line`
  escaping, template selection, and unchanged-header behavior are observable.
  C allocator failure, assertions, and malformed-input bugs are not proof of a
  portable specification; capture reference behavior before deciding fidelity
  policy. Milestone B compares grammar-stage snapshots and diagnostics, not
  the automaton or generated C. See the compatibility boundary for explicitly
  excluded undefined C behavior.

## Separate Go target — issue #13

`cmd/glcc` and `cmd/glemon` call the same `RunGo` in `internal/lemon/go.go`.
They do not select a language on `Run` or alter C options/template emission.
The new entry point reuses preprocessing, grammar preparation, FIRST/nullable,
LALR construction, conflict resolution, compression, state sorting, and numeric
packing directly. A Go-only scanner mode recognizes raw string literals and
diagnoses incomplete trailing productions/declarations; the C scanner path
retains its pinned behavior.

`go.go` emits typed semantic fields and per-rule Go locals, then renders the
embedded `lempar.go.tmpl` with `text/template` and formats the complete source
with `go/format`. Aliases are locals, never textual replacements. Destructor
`$$` substitution uses `go/scanner` to leave comments and literals untouched.
Runtime identifiers are namespaced by `%name`; reduction temporaries use the
reserved `_lemon` prefix, avoiding ordinary alias names. Semantic structs use
neither reflection nor dynamic assertions. Generated parsers have no dependency
on this module.

The runtime follows the pinned C stack and shift/reduce action ranges, including
pending SHIFTREDUCE actions, fallback before wildcard, and three-shift error
suppression. It clears popped semantic storage and supports fixed limits or a
growable Go slice. Actions own aliased RHS values; unaliased values are destroyed.
Unlike C union-copy semantics, actionless Go rules produce typed zero values.
Consequently Go disables only the no-code single-symbol bypass pass of table
compression, retaining actual reductions. The C emitter and pinned template
are unchanged. See the [Go contract](reference/go-parser.md) for declarations,
hooks, ownership, and intentionally different CLI/template APIs.

## Independent LL(1) generator — issue #14

`cmd/glime` calls `internal/ll1.Run`. `lemon.ParseGrammar` is the small shared
frontend boundary: preprocessing, Go-aware scanning, parsed symbol/rule identity,
source-order indices, and declaration locations, without `prepare` or LR work.
The old entry points are unchanged; declaration tracking is opt-in for this API.

`internal/ll1/analyze.go` computes independent nullable/FIRST/FOLLOW/SELECT fixed
points, nullable-prefix left-corner cycles, productivity and reachability.
Tables map a nonterminal and terminal to a source production; conflicts preserve
the first entry solely for explanation and prohibit emission. Stable grammar
slices determine all presentation and emitted table ordering.

`emit.go` and embedded `stack.go.tmpl` generate a standalone complete-input Go
function. A production frame owns typed RHS storage; a completion task runs its
action and writes its result into its parent's designated slot. A new expansion
never reuses an ancestor's storage. Aliases are Go locals, never rewritten text.
No LR state machine, conflict resolver, packed table, cgo or reflection is used.
See the [glime contract](reference/glime.md) for supported hooks and exclusions.

## Shared PL/0 example — issue #15

`examples/pl0/lalr/grammar.y` and `ll1/grammar.y` contain Go actions that
construct the same `lang.Block`, `Statement`, and `Expr` nodes. Both adapters
use `lang.Lex`, map symbolic terminal names to their own generated numeric IDs,
and return `(*lang.Block, error)`. The LALR frontend feeds EOF explicitly; the
predictive frontend omits it and uses the lexer's final position on EOF errors.
Only the LL(1) frontend owns expression-tail bookkeeping and folds it left;
the shared AST has no parser-specific nodes or evaluator closures.

`lang.Execute` first builds lexical scopes and validates every body, then
executes the AST. Procedure declarations retain lexical scopes; activations
have fresh variable maps and lexical parents. Runtime name lookup follows
those parents, not the call chain. Both frontends call this same interpreter;
`lang` imports neither parser nor generator internals. No generator algorithms,
templates, pinned reference files, or C output contracts change for the example.
