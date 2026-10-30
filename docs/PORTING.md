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
