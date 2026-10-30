# Milestone D table oracle

Run from the repository root, with no network access required:

```sh
scripts/test-tables.sh --update-goldens  # C-only regeneration + independent repeat
scripts/test-tables.sh --reference-only # rebuild C, compare goldens + repeat
scripts/test-tables.sh                  # rebuild C and Go, compare both + goldens
```

The differential command passes `--table-snapshot` to the Go executable.
Reference-only modes require a C compiler (`CC`, default `cc`), not Go. The
differential mode additionally requires the Go toolchain declared by `go.mod`.
All modes verify `reference/SHA256SUMS`; pinned sources are never modified.
The harness sets `LC_ALL=C`, `TZ=UTC`, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`,
`GOPROXY=off`, and `GOSUMDB=off`.

Each run uses isolated temporary directories and the same `grammar.y` input
filename and `./lemon` executable name. Every case has C-produced `stdout.txt`,
`stderr.txt`, and `exit-status.txt` goldens. Both `cmp` and `diff -bw` compare the
output inventory and every output file, including unexpected artifacts. No
output is normalized. Failures retain the complete temporary workspace, copied
inputs, executables, manifest, compiler/platform/environment record, and all
captures; the script prints that location and a reproduction command. Successful
runs remove temporary files. Regeneration does not delete stale golden files:
inventory checks must expose them.

## Boundary and record schema

`reference/table-snapshot.c` includes pristine `reference/sqlite/tool/lemon.c`
with its main renamed. Its own main mirrors the automaton driver and then the
original main's `CompressTables` (unless `-c`) and `ResortStates` (unless `-r`).
The numeric action ranges and the entire `ReportTable` axset/acttab packing and
`doesReduce` blocks are copied verbatim. No emitter, template processing,
`translate_code`, byte-size accounting, FIRST-set snapshot, or configuration
snapshot runs. These are intermediate table goldens, not generated-C goldens.

Records are tab-separated with a final newline, in this order:

```text
tables nsymbol nterminal nrule nstate nxstate nconflict
codes minShiftReduce errAction accAction noAction minReduce maxAction
bounds mnTknOfst mxTknOfst mnNtOfst mxNtOfst
sizes acttab_action_size acttab_lookahead_size
rule rule.index doesReduce
state statenum nTknAct nNtAct iTknOfst iNtOfst iDfltReduce autoReduce pDfltReduce.index-or--1
action sp.index type target spOpt.index-or--1 compute_action
entry i lookahead action
yy-action i value
yy-lookahead i value
yy-shift i offset
yy-reduce i offset
yy-default i value
```

`rule` records follow `lem.rule` order, not original index order. Every sorted
state is included, even tail states beyond `nxstate`; its actions immediately
follow it in existing list order. Action types are the pinned C enum integers.
Targets are state numbers for SHIFT/SH_RESOLVED/SSCONFLICT; original rule indices
for REDUCE/SRCONFLICT/RRCONFLICT/RD_RESOLVED/NOT_USED/SHIFTREDUCE; `-` otherwise.
Encoded actions and defaults use reordered `iRule`, not original `rule.index`.
`maxAction` is the original exclusive reduce-range upper bound.

`entry` records include every raw index below `pActtab->nAction`, preserving
`-1` holes. `sizes` gives the trimmed action size and raw lookahead size, not
the final padded lookahead length. Final arrays follow `ReportTable`:

* `yy-action`: trimmed action size, unused entries replaced with `noAction`.
* `yy-lookahead`: raw holes replaced with `nsymbol`, then appended `nterminal`
  entries until length is at least `nterminal + nactiontab`.
* `yy-shift`: only retained states, trailing `NO_OFFSET` removed, remaining
  missing offsets replaced with `nactiontab`.
* `yy-reduce`: only retained states, trailing `NO_OFFSET` removed, remaining
  missing offsets replaced with `mnNtOfst - 1`.
* `yy-default`: retained states only, `errAction` if `iDfltReduce < 0`, otherwise
  `minReduce + iDfltReduce`.

**Pinned `-r` behavior is intentionally preserved.** In this revision,
`ResortStates` also initializes action counts, offsets, defaults and `nxstate`.
Skipping it leaves `nxstate=0`, so packing/doesReduce visit no states, raw sizes
are zero, and the only final array records are `nterminal` padding lookaheads.
All states/actions are still snapshotted; uninitialized-by-Resort fields retain
their original zero-initialized values. The oracle does not repair this quirk
by calling part of ResortStates or introducing substitute initialization.

## Coverage and reference counts

There are **48 cases**: all **33 existing automaton cases**, six additional
`-c`/`-r` combinations (precedence and SQLite), and nine targeted cases using
five new grammars. Existing fixtures are referenced directly by
`../automaton/fixtures/...` relative to this directory, not copied. Feature,
omit, define/undefine, malformed, missing, empty, conflict, precedence,
fallback, wildcard and error cases are inherited. Invalid CLI-option testing
remains in the grammar harness because these instrumentation drivers expose
only a limited option set.

The targeted grammars verify:

* `no-code-chain.y` versus `empty-action-chain.y`: nonterminal SHIFTREDUCE
  bypass and `spOpt` chains, with absent actions versus explicit `{}` actions.
  The former bypasses rules 1 and 2 (`doesReduce=0`); the latter retains both.
* `wildcard-default.y`: a state containing a wildcard shift and an empty-rule
  reduction retains `iDfltReduce=-1`, even though another state defaults.
* `default-ties.y`: two reductions each occur twice. The first lookahead's
  rule (original index 5) wins over the lower rule index 4; suppressed actions
  remain as NOT_USED records.
* `error-shiftreduce.y`: `error` gets `minShiftReduce + iRule` while an ordinary
  nonterminal gets `minReduce + iRule`; explicit actions prevent bypass.

Verified C counts (action/lookahead sizes are raw `sizes` fields):

| SQLite case | Symbols | Terminals | Rules | States | Retained | Action size | Raw lookahead size | Final lookahead length |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| default / `-l` | 323 | 187 | 409 | 845 | 583 | 2207 | 2341 | 2394 |
| features | 324 | 188 | 411 | 857 | 594 | 2387 | 2416 | 2575 |
| omit | 274 | 172 | 317 | 628 | 430 | 1879 | 1922 | 2051 |
| undefine | 295 | 174 | 374 | 769 | 535 | 1922 | 2078 | 2096 |
| `-c` | 323 | 187 | 409 | 845 | 845 | 17281 | 17337 | 17468 |
| `-r` / `-c -r` | 323 | 187 | 409 | 845 | 0 | 0 | 0 | 187 |

All SQLite variants have zero conflicts. Default SQLite has 12,681 action
records, 583 final shift offsets, 413 final reduce offsets, and 583 defaults.
These measurements describe the intermediate boundary before code translation,
not a claim of full parser-generation compatibility.
