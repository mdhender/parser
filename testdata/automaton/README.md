# Milestone C automaton oracle

The separate `reference/automaton-snapshot.c` executable includes the pristine,
pinned `reference/sqlite/tool/lemon.c` unchanged. Initialization matches
`reference/grammar-snapshot.c`; execution follows upstream `main` through
`FindActions`, stopping **before** `CompressTables`, `ResortStates`, reports,
tables, or generated C. This is not a parser-generation compatibility claim.

## Reproduce

From the repository root:

```sh
scripts/test-automaton.sh --reference-only
scripts/test-automaton.sh --update-goldens  # explicit C-only regeneration
scripts/test-automaton.sh differential    # default; Go --automaton-snapshot
```

All modes verify `reference/SHA256SUMS` before building. The reference remains
SQLite 3.50.4, Fossil check-in
`4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3`.
The manifest checks the parser source, template, SQLite grammar, build recipe,
revision identifier and version; no downloads occur. See `docs/REFERENCE.md`
for provenance and the pinned hashes.

Build: `${CC:-cc} -o snapshot reference/automaton-snapshot.c` (no added compiler
flags). Runs use isolated directories, the identical input name `grammar.y`,
and executable name `./lemon`. The Go run adds `--automaton-snapshot` ahead of
the same manifest arguments. `LC_ALL=C`, `TZ=UTC`, `CGO_ENABLED=0`,
`GOPROXY=off`, `GOSUMDB=off`, and `GOTOOLCHAIN=local` are set explicitly.
The harness requires its compiler and comparison tools, and Go only in
differential mode; a missing required tool is an error, never a skip.

The four-column manifest is `name`, `automaton`, `input`, `options` (tabs).
Inputs are fixture basenames, `sqlite` for the pinned `parse.y`, or `missing`.
Option words contain no whitespace and are never evaluated as shell code.
Invalid-option diagnostics remain in the grammar harness: the instrumentation's
intentionally restricted `D`, `U`, `l` help table is not upstream's full CLI.

Every case compares output inventories, stdout, stderr, status and any generated
files, both byte for byte and with `diff -bw`. No differences are permitted or
normalized. Both C-only modes run each case twice independently. Failures keep
the temporary directory, binaries, copied inputs, captured outputs, case
manifest and environment/tool-version log, and print the reproduction command.
Abnormal termination is rejected rather than recorded as golden behavior.

## Snapshot schema

Each record ends in a newline; fields are tab-separated decimal integers unless
specified otherwise. Empty final fields retain the final tab.

```text
automaton  nsymbol  nterminal  nrule  nstate  nconflict
first      symbol.index  lambda  terminal-index-set
reduce     rule.index  lhsStart  canReduce
state      statenum
basis      rule.index  dot
config     rule.index  dot  follow-index-set
action     symbol.index  type  target
```

There is no grammar snapshot prefix. `first` includes every ordinary symbol
`i < nsymbol` in indexed order, not `{default}` or named multiterminals. Nullable
flags are 0/1; terminals have empty FIRST sets. FIRST members range from 0
through `nterminal-1`. Follow members range from 0 through **`nterminal`
inclusive**, matching the upstream set allocation, not only the reduction loop.
Sets contain comma-separated ascending indices with no spaces.

`reduce` records follow `lem.rule` order after `Rule_sort`, but use the original
`rule.index`, not `iRule`. States retain discovery order and original numbers.
Basis and closure lists are the upstream sorted `(rule.index, dot)` lists.
Actions retain their final linked-list order after conflict resolution; do not
re-sort them by their now-mutated action types.

Action enum and target:

| Type | Integer | Target |
| --- | ---: | --- |
| SHIFT | 0 | original state number |
| ACCEPT | 1 | `-` |
| REDUCE | 2 | original rule index |
| ERROR | 3 | `-`, even when its union still contains a SHIFT pointer |
| SSCONFLICT | 4 | original state number |
| SRCONFLICT | 5 | original rule index |
| RRCONFLICT | 6 | original rule index |
| SH_RESOLVED | 7 | original state number |
| RD_RESOLVED | 8 | original rule index |
| NOT_USED | 9 | `-` (not produced at this boundary) |
| SHIFTREDUCE | 10 | `-` (not produced at this boundary) |

Parse failures return their parser error count and do not emit a snapshot, as
in the grammar instrumentation. Empty grammar returns 1 without a snapshot.
After successful parsing, even automaton diagnostics do not suppress snapshots.
Final status is boolean `(errorcnt > 0 || nconflict > 0)`. Conflicts additionally
emit `%d parsing conflicts.\n` to stderr. Upstream unreducible-rule diagnostics
include a second newline, which is preserved verbatim in the goldens.

## Ordering hazard: exact action comparator

The pinned `actioncmp` compares symbol index, then numeric action type, then
original rule index only for REDUCE/SHIFTREDUCE. If still tied, its final test is
exactly **`(int)(ap2 - ap1)`**, not stable insertion order, state number, or rule
number. `Action_new` allocates slabs of 100 actions and returns ascending slots.
Within one slab, later-created duplicate shifts therefore sort before earlier
ones. An independent probe against the pristine include observed
`actioncmp(older,newer) == 1`, its reverse `-1`, and shift targets sorted `2,1`
when created in order `1,2`.

`ss-overlap` and `ss-ordered` exercise that real ordering without replacing the
comparator: the later rule's shift to state 3 survives; the earlier rule's shift
to state 1 becomes SSCONFLICT. Ordered constituents matter: `A|B` and `B|A`
are distinct to `same_symbol`, yielding two SS conflicts in `ss-ordered`.

**Unresolved portability limit:** subtraction of action pointers from different
allocation slabs is undefined in ISO C, and the cast to int adds another hazard
for large differences. A global reverse creation counter is not proven
equivalent across slabs/allocators/platforms. These small duplicate-shift
fixtures stay within one slab; no cross-slab tie portability claim is made.
The oracle deliberately preserves, rather than hides, the pinned comparator.

## Coverage and recorded validation

Fixtures cover nullable cycles; suffix FIRST including nullable suffixes and
multiterminals; canonical LR(1)-not-LALR(1) merged lookaheads; every SR precedence
outcome (missing rule precedence, unranked shift, unranked rule, higher/lower
precedence, LEFT, RIGHT, NONE); RR equal/unranked unresolved and both precedence
winner directions; overlapping ordered multiterminals; fallback/wildcard/error;
explicit/missing/RHS start; undefined nonterminal; action-driven rule ordering;
preprocessing D/U/l; malformed, empty and missing inputs; and SQLite variants.

The RHS-start fixture uses right recursion. An initial left-recursive version
triggered an upstream SHIFT/ACCEPT assertion in `resolve_conflict`; it was
replaced before golden generation. This crashing grammar is not in the suite.
An undefined nonterminal and an unreachable rule intentionally retain their
automaton snapshots plus diagnostics. RR precedence resolution can remove all
reductions of a losing rule: zero conflicts does not necessarily mean status 0.

Verified on Linux x86-64 with Debian GCC/cc 12.2.0 and Go 1.27.3:

- `bash -n scripts/test-automaton.sh`: passed.
- `scripts/test-automaton.sh --update-goldens`: generated C goldens and repeated
  all 33 final cases with byte and `diff -bw` equality.
- `scripts/test-automaton.sh --reference-only`: passed with independent repeats.
- `scripts/test-automaton.sh differential`: all 33 final cases passed against Go.

The final suite has 25 fixture grammars, 33 case directories, and 99 golden
files. There are 30 snapshots (three inputs fail before the automaton), 14
successful statuses and 19 expected status-1 cases. Snapshots collectively
contain 4,057 states, 11,560 basis records, 37,451 configurations and 59,801
actions. Types 0 through 8 all occur, including one ERROR and three SSCONFLICTs.

SQLite counts before compression/resorting:

| Case | Symbols | Terminals | Rules | States | Conflicts |
| --- | ---: | ---: | ---: | ---: | ---: |
| default / no-lines | 323 | 187 | 409 | 845 | 0 |
| features | 324 | 188 | 411 | 857 | 0 |
| omit | 274 | 172 | 317 | 628 | 0 |
| undefine | 295 | 174 | 374 | 769 | 0 |

The LR(1)-not-LALR(1) fixture has 12 states and two RR conflicts. The nullable
cycle has five states and one SR conflict. The overlap fixtures have five states
and one/two SS conflicts. The precedence fixture has nine states, no conflicts,
and includes the nonassociative ERROR action with a `-` target.
