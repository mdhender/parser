# Porting log

## Milestone A — Pinned C reference baseline

- Reference: SQLite 3.50.4, Fossil check-in
  `4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3`.
- Sources: pristine `lemon.c`, `lempar.c`, `parse.y`, and supporting provenance
  files under `reference/sqlite/`; SHA-256 manifest under `reference/`.
- C functions translated: **none**. There is no Go generator yet.
- Go files: `reference/reference_test.go` checks fixture integrity offline.
- Tooling: `scripts/generate-reference.sh` builds/runs the oracle;
  `scripts/compare-artifacts.sh` compares files and process results;
  `scripts/test-reference.sh` independently rebuilds/reruns twice and compares
  against the goldens and between runs.
- Configuration and artifact hashes: [REFERENCE.md](REFERENCE.md).
- Initial subsystem analysis: [ARCHITECTURE.md](ARCHITECTURE.md).
- Compatibility scope and unimplemented features: [COMPATIBILITY.md](COMPATIBILITY.md).
- Intentional deviations from upstream: none in vendored source or generated
  outputs. The scripts isolate directories, fix locale/time zone, and capture
  diagnostics/metadata without changing the reference code.
- The baseline includes `parse.sql` because upstream's `-S` enables it, in
  addition to `parse.c`, `parse.h`, and `parse.out`.
- Validation: `scripts/test-reference.sh` passed, reproducing all four generated
  files plus stdout, stderr, and exit status byte for byte in two independent
  builds/runs. All generated files also passed `diff -bw`.
- Offline Go validation: `go test -count=3 ./...` and `go vet ./...` passed with
  `GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0`; the repeated test also used a
  nonexistent `CC` to check that ordinary tests do not need a C compiler.
- Negative validation in disposable copies: changed source/golden hashes,
  each changed output/process result, missing artifacts, and whitespace-only
  output changes were rejected. Missing compiler checks failed explicitly and
  retained their work directory. Generation/comparison with spaces in the
  output path passed; reusing an existing output directory was rejected.
- Unresolved work: Go translation and C-versus-Go differential validation have
  not started. No SQLite runtime/build validation is claimed.
- Next recommended task, **only after authorization**: Milestone B, starting
  with command-line processing, byte-oriented grammar scanning/preprocessing,
  symbol/rule representation, and small grammar fixtures. Compare observable
  preprocessed/reprinted grammar output with the pinned C oracle before moving
  to automaton construction.

## Milestone B — Grammar processing

Authorized by the owner's explicit request to implement Milestone B. All 6,056
lines of the pinned `lemon.c` were read before completing translation; no other
Go port or alternate parsing algorithm was used. Milestone A's sources, pin,
configuration, and final parser goldens remain unchanged.

- Translated C routines: grammar-stage `main`; `OptInit`, `handleflags`,
  `handleswitch`, `OptNArgs`/`argindex`, `errline`, `OptPrint`; D/U callbacks;
  `eval_preprocessor_boolean`, `preprocess_input`, `Parse`, `parseonetoken`;
  `Symbol_new`/lookup/insertion-order enumeration, `Symbolcmpp`; rule numbering
  and `Rule_sort` ordering; `FindRulePrecedences`; `Reprint`/`rule_print`;
  grammar diagnostics and declaration `#line` construction.
- Implementation: `cmd/lemon/main.go`; `internal/lemon/{lemon,options,grammar,
  preprocess,scanner,parser,output}.go`. Subsystem ownership and representation
  choices are recorded in [ARCHITECTURE.md](ARCHITECTURE.md).
- Public command modes: `-E`, `-g`, `-x`, and project-only
  `--grammar-snapshot`. Normal generation rejects explicitly and creates no
  artifacts. Generation-only flags have no effect in grammar-only modes;
  their actual behavior remains unimplemented.
- Offline tests: `internal/lemon/lemon_test.go` executes the public command API
  against 58 C-generated process-output/snapshot cases and asserts no generated
  files. Additional tests cover both sides of the 1000-symbol RHS boundary and
  explicit unsupported-generation behavior. Expectations are never derived from
  the Go implementation.
- Differential tooling: separate `reference/grammar-snapshot.c` includes the
  pristine C source and stops after precedence assignment. The instrumented
  driver is never used for final parser comparisons. `scripts/test-grammar.sh`
  builds it, pristine C, and Go; compares C to saved goldens and Go to C in
  equivalent isolated paths; checks bytes, `diff -bw`, stdout, stderr, status,
  and complete output inventories; retains full failures for reproduction.
- All declarations are exercised by small grammar fixtures. Cases also cover
  aliases, action text, empty versus missing values, rule/source ordering,
  fallback/wildcard, named/anonymous compounds, precedence and NEVER-REDUCE,
  raw UTF-8 code, byte-based identifiers, error recovery in grammar parsing,
  scanner/preprocessor errors, repeated destructors, nested conditions,
  whole-expression short-circuiting, duplicate defines, D/U ordering, and CLI
  parsing quirks/diagnostics.
- SQLite default snapshot matches **323 normal symbols, 187 terminals, 409
  rules, and 344 explicit-action rules**, as well as every serialized field.
  Further snapshots match with line suppression, enabled optional features,
  omitted features, and define/undefine combinations. This establishes grammar
  representation equivalence for these configurations, not parser equivalence.
- Executed validation: `scripts/test-grammar.sh` passes all 58 cases byte for
  byte and under `diff -bw`; reference-only repeats reproduce their goldens.
  `scripts/test-reference.sh` still reproduces every Milestone A artifact in
  two independent builds/runs. `GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0
  CC=/nonexistent/compiler go test -count=3 ./...` and offline `go vet ./...`
  pass using Go 1.27.3.
- Additional disposable validation: 272 expression/define/mode combinations
  match pristine C stdout/stderr/status byte for byte, including malformed and
  incomplete expressions and short-circuit boundaries. Both 1000 and 1001 RHS
  symbols also match pristine C. A `CGO_ENABLED=0` built executable is reported
  statically linked by `file`; `ldd` reports `not a dynamic executable`.
- Intentional deviations: per-invocation ownership and immutable Go text replace
  C statics/allocation pools/interning; original ordering and nil/empty semantics
  remain. The inspection option and explicit generation failure mark the stage
  boundary. Go does not reproduce undefined C out-of-bounds reads or null
  callbacks. Detailed exclusions are in [COMPATIBILITY.md](COMPATIBILITY.md).
- Unresolved grammar differences in executed comparisons: **none**. No FIRST
  sets, automaton, conflicts, action tables, C emission, or SQLite runtime
  compatibility is claimed. Aliases and C hooks are stored, not rewritten or
  executed. The differential command requires C; ordinary Go tests do not.
- Next recommended task, **only after authorization**: Milestone C. Begin with
  `FindFirstSets` and the byte-set routines, then configuration closure/state
  construction, propagation links/lookahead, and conflict resolution; extend
  snapshots to find the earliest intermediate divergence. Do not begin table
  packing or C emission ahead of their milestones.

## Milestone C — Parser automaton

Authorized by the owner's explicit request to implement Milestone C. The pinned
C source, reference invocation, and Milestone A final parser goldens are
unchanged. No table packing, compression, state renumbering, or C emission was
implemented.

- Translated C routines: `SetSize`/`SetNew`/`SetAdd`/`SetUnion`, `FindFirstSets`,
  `Configlist_add`/`Configlist_addbasis`/`Configlist_closure` and sorted builder
  extraction, `Configcmp`, structural state lookup, `FindStates`, `getstate`,
  `same_symbol`, `buildshifts`, `FindLinks`, `FindFollowSets`, `Action_add` and
  action sorting, `FindActions`, and `resolve_conflict`.
- Go ownership: `internal/lemon/automaton.go` contains byte sets, nullable/FIRST,
  configuration builders, state graph, and propagation; `action.go` contains
  original enum values, action ordering, and pairwise conflict resolution.
  `grammar.go` owns the new graph/metadata fields. `output.go`, `options.go`,
  and `lemon.go` expose the project-only `--automaton-snapshot` command.
- The inspection mode stops after `FindActions`, preserves all conflict and
  precedence-resolution markers, and serializes original rule IDs and discovery
  state IDs. It prints the graph even after automaton errors and returns 1 for
  errors or conflicts. Ordinary generation still fails explicitly with no
  artifacts. Existing `-E`, `-g`, and grammar snapshot behavior is unchanged.
- Reference tooling: `reference/automaton-snapshot.c` includes pristine C and
  mirrors initialization through `FindActions`; `scripts/test-automaton.sh`
  rebuilds it and Go offline and compares saved C expectations, C output, and Go
  output in equivalent isolated paths. Both bytes and `diff -bw`, diagnostics,
  status, and complete output inventories must match. Failed runs are retained.
- Offline public-command tests reuse the existing oracle runner and checked-in
  automaton fixtures. Cases distinguish nullable-prefix/suffix FIRST, recursion,
  LALR basis merging versus canonical LR(1), all precedence/associativity branches,
  unresolved SR/RR/SS conflicts, ordered multiterminals, reserved `error`,
  fallback/wildcard metadata, start selection, unreachable/undefined rules,
  generated versus original rule order, malformed input, and SQLite variants.
- Executed validation: all **33 automaton cases** match C byte for byte and with
  `diff -bw`; `--reference-only` independently repeats/reproduces all goldens.
  Default SQLite matches **323 normal symbols, 187 terminals, 409 rules,
  845 uncompressed states, zero conflicts**, including every FIRST/nullable set,
  basis/closure item, configuration lookahead, action, and reducibility flag.
- Existing validation remains green: all **58 grammar differential cases**;
  `scripts/test-reference.sh` reproduces all final C artifacts and process
  results in two independent builds/runs. `GOTOOLCHAIN=local GOPROXY=off
  CGO_ENABLED=0 CC=/nonexistent/compiler go test -count=3 ./...` and offline
  `go vet ./...` pass on Go 1.27.3. An explicitly missing compiler makes the
  automaton differential script fail clearly with status 2, not skip checks.
  A `CGO_ENABLED=0` executable is statically linked according to `file`.
- Intentional representation deviations: recursive local builders replace C
  statics, slices/maps replace allocation pools and hash buckets, ordered slices
  supply all enumeration, and slice propagation edges reach the same fixed
  point. Reverse action-creation order replaces the C pointer tie-break; C's
  cross-allocation pointer subtraction is undefined and not a portable contract.
  See [COMPATIBILITY.md](COMPATIBILITY.md) for this exclusion and stage boundaries.
- Unresolved differences in executed comparisons: **none**. No packed-table,
  report/header/C/SQL output or SQLite runtime compatibility is claimed.
- Next recommended task, **only after authorization**: Milestone D, translating
  `CompressTables`, `ResortStates`, `compute_action`, and packed action-table
  placement/offset construction, with new intermediate reference snapshots.

## Milestone D — Table generation

Authorized by the owner's explicit request to implement Milestone D. The pinned
C sources, template, reference configuration, and Milestone A parser goldens
remain unchanged. Work stops at numeric tables; no C emitter was implemented.

- Translated routines: `CompressTables` (all three passes), `ResortStates` and
  `stateResortCompare`, `compute_action`, `acttab_alloc`/`acttab_action`/
  `acttab_insert`/`acttab_action_size`, `axset_compare`, and the numeric action
  ranges, packing, post-optimization `doesReduce`, and five-array preparation
  portions of `ReportTable`.
- Go implementation: `internal/lemon/table.go`; graph fields in `grammar.go`
  and `automaton.go`; shared action comparator and NOT_USED/SHIFTREDUCE in
  `action.go`; project-only `--table-snapshot` in `options.go`/`lemon.go`.
  Original rule IDs, generated rule IDs, full states, and retained states remain
  distinct. Numeric arrays are available for the future emitter without C text.
- Preserved ordering: first encountered most-frequent default reduction, state
  zero pinned first, descending nonterminal/terminal counts then descending old
  state number, largest action sets first with original transaction-order ties,
  backward exact-transaction reuse followed by forward collision-free holes.
  Missing-lookahead rejection, negative nonterminal offsets, safe terminal
  placement, raw holes, action trimming, and lookahead padding follow C.
- `-c` skips optimization but still packs. `-r` skips the whole `ResortStates`
  call, intentionally retaining C's zero initialized `nxstate` and empty packed
  action tables rather than silently initializing a repaired parser.
- Oracle: separate `reference/table-snapshot.c` includes pristine C and copies
  the original packing/doesReduce block verbatim; no template processing or
  `translate_code` runs. `scripts/test-tables.sh` checks saved C expectations,
  fresh C, and Go with equivalent paths/options; stdout, stderr, status, output
  inventory, bytes, and `diff -bw` must match. All failure artifacts are retained.
- Offline public-command tests extend the existing oracle runner. The matrix
  has **48 cases**, including all 33 automaton inputs/configurations, `-c`/`-r`
  combinations, absent versus explicit-empty action chains, wildcard default
  suppression, frequency ties opposing rule order, and `error` versus ordinary
  nonterminal SHIFTREDUCE encoding. Every optimized state/action, rule-use flag,
  numeric range, offset, raw packed entry, and final numeric array is compared.
- Executed results: all **48 differential cases** match byte for byte and with
  `diff -bw`; C-only generation and independent reference-only repeats reproduce
  all goldens. Default SQLite matches **845 full states, 583 retained states,
  2,207 action entries, 2,341 raw lookahead entries**, and **2,394 padded
  lookahead entries**. With `-c`: 845 retained states and 17,281 action entries.
  Feature/omit/undefine variants match in full; all have zero conflicts.
- Independent check: every value in the five default-SQLite Go arrays matches
  the arrays extracted from the pristine Milestone A `parse.c` golden: action
  2,207; lookahead 2,394; shift offsets 583; reduce offsets 413; defaults 583.
  This checks numeric contents, not generated-C formatting or runtime behavior.
- Disposable validation: 24 seeded reachable acyclic grammars, with shuffled
  rule/token order, nullable alternatives, and mixed missing/empty actions,
  match C stdout/stderr/status under all four default/`-c`/`-r` combinations
  (**96 successful table comparisons**). No oracle input crashed or stopped at
  input rejection. An explicitly missing compiler makes the differential
  command fail with status 2 instead of skipping.
- Regression checks: all **58 grammar** and **33 automaton** differential cases
  remain exact; `scripts/test-reference.sh` reproduces every final C artifact
  and process result in two independent builds/runs. Offline `go test -count=3
  ./...` and `go vet ./...` pass with Go 1.27.3, `GOTOOLCHAIN=local`,
  `GOPROXY=off`, `CGO_ENABLED=0`, and a nonexistent `CC` for the tests. The Go
  executable is statically linked (`file`; `ldd`: not a dynamic executable).
- Intentional representation changes: numeric table arrays are prepared
  independently of C rendering; Go slices replace allocations while preserving
  the original acttab allocation/search bounds. Existing action-pointer-tie
  portability exclusions still apply. No unexplained differences were observed.
- Next recommended task, **only after authorization**: Milestone E. Translate
  template lookup/copying, output naming/report/header/SQL generation, semantic
  type selection, alias/destructor/action rewriting, and C emission using pinned
  `lempar.c`. Include C type-width choices and byte-size statistics; compare
  complete artifacts against uninstrumented C. Full parser/runtime compatibility
  and SQLite integration remain unclaimed.

## Milestone E — C code generation

Authorized by the owner's explicit request to implement Milestone E. Pinned
sources, template, invocation, build configuration, and all Milestone A goldens
remain unchanged. No other Go port or alternative algorithm was used.

- Translated routines: `file_makename`/`file_open`, `pathsearch`, `tplt_open`,
  `tplt_xfer`/`tplt_skip_header`, `tplt_linedir`/`tplt_print`, `RulePrint`,
  `PrintAction`, `ReportOutput`, `writeRuleText`, `ReportHeader`, `has_destructor`,
  `translate_code`/string construction, `emit_code`, `emit_destructor_code`,
  `print_stack_union`, `minimum_size_type`, and the remaining `ReportTable`
  macro/array/fallback/name/rule/destructor/action/hook/SQL rendering and byte
  accounting. Normal `main` generation now reaches the full pipeline.
- Go files: `template.go`, `code.go`, `emit.go`, `report.go` in `internal/lemon`;
  integration/metadata in `lemon.go`, `options.go`, and `grammar.go`. Numeric
  packing remains owned by the unchanged `BuildTables`; snapshot contracts
  remain unchanged. Uses standard library only, no cgo/C subprocesses.
- Preserved details: 999-byte template chunks, exact parser-name boundaries,
  header-skip and uncounted-ifdef line-counter quirks, used-define prologue,
  hashed semantic union slots, alias/major-code/overwrite rewriting and errors,
  temporary LHS storage, destructor prefixes/suffixes, shared switch cases,
  unchanged-header timestamps, input-relative names, no automatic directory
  creation, `.sql` line-restoration paths, partial artifacts on errors, `-r`'s
  zero retained states, and exact type/byte-size statistics.
- Tests: **52 full generation cases** in `testdata/generation/cases.tsv` use
  uninstrumented pinned C. The explicit `scripts/test-generation.sh` verifies
  C-derived saved hashes and compares full C/Go stdout, stderr, status,
  inventories, bytes, and `diff -bw`, retaining failures. Offline public-command
  tests use compact C-only artifact inventories/hashes; no new large duplicate
  SQLite goldens. They also check unchanged/stale header behavior.
- Executed results: all 52 cases match byte for byte and under `diff -bw`.
  C-only regeneration/repeats and independent reference-only execution reproduce
  all expectations. The exact `./lemon -S parse.y` case reproduces all four
  unchanged Milestone A artifacts, including `parse.c` SHA-256
  `77ff8fda334c8b0cc01f5734765f994417ba81417d50dedf3d0d7241fe8a4a52`.
  Default SQLite statistics match 583 retained states, 2,207 action entries,
  2,341 raw lookahead entries, and 12,628 accounted table bytes.
- Supplementary generated-C execution: C- and Go-generated arithmetic parsers
  compile and run; `7 - 2 * 3` returns value 1, one acceptance, no syntax errors.
  This is not SQLite compilation or runtime integration.
- Disposable differential validation: 12 additional cases match full artifacts
  and process results: type-width transitions near 127/255 symbols, 80 distinct
  semantic types/collisions, 999-byte template/name boundaries, nested/backslash
  input names, output-directory basename handling, and SQL-open partial failure.
- Regression checks: all 58 grammar, 33 automaton, and 48 table differential
  cases still pass; `scripts/test-reference.sh` reproduces all pristine C
  artifacts/process results twice. Offline `go test -count=3 ./...` and
  `go vet ./...` pass on Go 1.27.3 with `GOTOOLCHAIN=local`, `GOPROXY=off`,
  `CGO_ENABLED=0`, and nonexistent `CC` for tests. Missing compiler makes the
  generation script fail explicitly with status 2. `file` reports a statically
  linked Go binary; `ldd` reports no dynamic executable.
- Intentional representation differences: Go builders replace `append_str`
  storage, value equality replaces C interned-string pointer comparison, and
  output counting is local to an invocation. Existing undefined-C exclusions
  remain; no observed output differences are normalized or unexplained.
- Remaining verification limits: PATH-only lookup, permission-error distinctions,
  larger integer-width boundaries, exhaustive parser runtime/error recovery,
  and SQLite integration are not covered by the executed matrix. See
  [COMPATIBILITY.md](COMPATIBILITY.md) for the precise boundary.
- Next recommended task, **only after authorization**: Milestone F, a final
  compatibility audit, broader generated-runtime checks and SQLite source-tree
  compilation/testing where feasible. Milestone E is complete; final project
  compatibility certification remains open.
