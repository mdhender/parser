Project: Port SQLite’s Lemon Parser Generator from C to Go

Objective

Create a faithful, standalone Go port of SQLite’s Lemon parser generator.

The Go implementation must accept SQLite’s src/parse.y grammar and generate C parser artifacts that match the output of the original C implementation.

Primary acceptance criterion: Given the same SQLite source revision, grammar, template, command-line options, and output paths, the generated parse.c must match the reference implementation under diff -bw, with no unexplained differences.

The generated parse.h and parse.out should also match when produced by the reference command.

This is a translation project, not a redesign. Preserve the original implementation’s behavior, algorithms, ordering, and output conventions. Do not substitute alternative parsing algorithms simply because they are more idiomatic in Go.

The resulting program is written in Go but generates C parsers, just as the original Lemon does.

Constraints

* Go standard library only.
* No cgo or C dependencies in the final Go executable.
* The reference C implementation may be compiled and executed in tests.
* Preserve the original algorithms and observable behavior.
* Prefer straightforward, explicit Go code over abstractions.
* Avoid unnecessary interfaces, reflection, goroutines, and dependencies.
* Use gofmt and conventional Go naming where they do not obscure the relationship to the C source.
* Retain recognizable names for algorithms, structures, and functions to facilitate source comparison.
* Use Go’s built-in testing package.
* Do not use existing Go ports of Lemon as implementation sources. They may be consulted for research, but SQLite’s C implementation is authoritative.

1. Establish the Reference Implementation

Use SQLite’s official source repository:

https://sqlite.org/src

The relevant files are:

* tool/lemon.c
* tool/lempar.c
* src/parse.y

Pin all three files to the same immutable SQLite source revision. Record the source revision identifier and SHA-256 hashes in docs/REFERENCE.md.

Do not silently update the reference revision during development.

Compile the original Lemon implementation using the available C compiler.

Determine the exact invocation SQLite uses to generate its parser, including relevant options, preprocessor definitions, template selection, and working directory.

Select a specific SQLite build configuration at the pinned revision. Preserve the relevant build-rule excerpt from that revision and record the resolved feature flags, compiler command and version, platform, and environment settings that affect generation in docs/REFERENCE.md.

Preserve the reference build and parser-generation invocations in reproducible scripts.

Generate the baseline artifacts:

* parse.c
* parse.h
* parse.out, if enabled by the selected invocation

Store the baseline artifacts as golden fixtures and provide a deterministic script that regenerates them from the pinned reference sources.

Ensure both implementations receive equivalent input paths so that embedded filenames and #line directives do not cause artificial differences.

Required milestone

Before implementing the Go port, demonstrate that rebuilding and rerunning the original C implementation reproduces the same golden files byte for byte.

2. Study and Document the Original Implementation

Read lemon.c in full before translating its major subsystems.

Create docs/ARCHITECTURE.md documenting:

* Main execution flow
* Grammar scanning and preprocessing
* Symbol and rule representations
* Symbol tables and lookup behavior
* State and configuration construction
* FIRST and nullable calculations
* Lookahead propagation
* Conflict resolution
* State ordering and renumbering
* Action-table construction and compression
* Template processing and output generation

Identify global state, linked structures, allocation ownership, comparators, and routines whose behavior depends on iteration or insertion order.

Record important C-to-Go translation hazards, including:

* Integer widths and conversions
* Signed versus unsigned comparisons
* Pointer identity
* Linked-list traversal order
* Hash-table iteration
* Sorting stability and comparator behavior
* Null pointers versus zero values
* Buffer and string handling
* Character classification
* Output formatting

Do not assume Go map iteration order is suitable where the original implementation depends on deterministic traversal.

3. Port Incrementally

Organize the implementation into logical subsystems corresponding to the original C program.

A suggested layout is:

cmd/lemon/
    main.go
internal/lemon/
    lemon.go
    scanner.go
    grammar.go
    symbol.go
    rule.go
    config.go
    state.go
    action.go
    table.go
    report.go
    template.go
    output.go
reference/
    ...
testdata/
    grammars/
    golden/
docs/
    REFERENCE.md
    ARCHITECTURE.md
    PORTING.md
    COMPATIBILITY.md

This layout is a starting point, not a requirement. Preserve clear subsystem boundaries without introducing unnecessary architecture.

Translate one subsystem at a time.

For each subsystem:

1. Identify the corresponding C functions and data structures.
2. Document any behavior that must be preserved.
3. Translate the implementation into Go.
4. Add focused tests.
5. Compare intermediate behavior against the C reference where practical.
6. Record progress and unresolved differences.

Avoid large, unverified translation batches.

Translation principles

Preserve the original data flow and algorithmic choices.

Do not replace Lemon’s table construction, state ordering, or compression routines with independently designed equivalents.

Avoid optimizing the implementation until compatibility has been demonstrated.

When a direct translation looks awkward in Go, prefer the awkward but easily verified translation initially.

Use comments to explain departures from the original C implementation.

4. Build a Differential Test Harness

Implement a test harness that runs the C and Go implementations against identical grammars.

The harness must:

* Use isolated temporary output directories.
* Pass equivalent options to both implementations.
* Capture stdout, stderr, and exit status.
* Compare stdout, stderr, and exit status, including for malformed grammars and invalid command-line options. Document any permitted diagnostic differences in docs/COMPATIBILITY.md.
* Compare generated artifacts.
* Report the first meaningful difference.
* Retain full output artifacts when a comparison fails.
* Make failures reproducible.

Support both strict byte-for-byte comparison and whitespace-insensitive comparison equivalent to diff -bw.

Run both comparisons for every generated artifact. Review and document every byte-level difference, even when diff -bw passes. Whitespace-insensitive equality alone does not establish semantic equivalence: it can conceal changes inside C string literals and preprocessor definitions. Only documented formatting differences that preserve behavior are permitted.

Do not normalize, reorder, or otherwise transform generated output to conceal differences.

Tests must distinguish between expected formatting differences and actual compatibility failures.

Use the original C implementation as the oracle.

Intermediate comparisons

Where useful, instrument a reference build of lemon.c to emit deterministic snapshots of internal state.

Possible snapshots include:

* Parsed grammar rules
* Symbol numbers and classifications
* FIRST sets
* Parser states and configurations
* Lookahead sets
* Shift/reduce and reduce/reduce conflicts
* Action tables
* Compressed table offsets and entries

The instrumentation must not change the behavior of the reference implementation.

Keep the pinned reference sources pristine. Apply instrumentation only to separate copies, and use an uninstrumented reference build for final artifact comparisons.

Compare corresponding Go snapshots against the C snapshots.

These comparisons are particularly important because two correct parser generators can produce different state numbering and table layouts.

5. Implement Compatibility in Stages

Milestone A — Reference baseline

Deliver the pinned SQLite reference sources, build scripts, golden files, and reproducibility tests.

Milestone B — Grammar processing

Port scanning, preprocessing, declarations, rules, precedence, and symbol handling.

Verify that SQLite’s parse.y produces equivalent internal grammar representations.

Milestone C — Parser automaton

Port configuration closure, state construction, lookahead propagation, and conflict resolution.

Verify state counts, symbol numbering, configurations, and actions against the C reference.

Milestone D — Table generation

Port state sorting, renumbering, action-table construction, and compression.

Verify the resulting tables and offsets against the reference implementation.

Milestone E — C code generation

Port template processing and C output generation.

Use the pinned original lempar.c template.

Generate parse.c, parse.h, and parse.out as appropriate.

Milestone F — Final compatibility

Run the Go implementation against SQLite’s pinned parse.y using the reference invocation.

The primary acceptance test must pass:

diff -bw reference-output/parse.c go-output/parse.c

Also compare parse.h and parse.out whenever they are produced.

Investigate every discrepancy. Do not mark the milestone complete merely because the generated C compiles or successfully parses SQL.

6. Validation

In addition to output comparisons:

* Run `go test ./...`.
* Run `go vet ./...`.
* Run tests repeatedly to detect nondeterministic output.
* Test several smaller Lemon grammars before SQLite’s full grammar.
* Exercise precedence, fallback tokens, wildcard tokens, error recovery, and relevant grammar directives.
* Verify error handling for malformed input.
* Confirm the Go executable can run without a C compiler or C runtime dependency beyond normal operating-system facilities.

Ordinary go test ./... must run offline without a C compiler, using Go tests and checked-in golden fixtures. Provide and document a separate explicit command for reference reproducibility and C-versus-Go differential tests. That command must fail clearly when a C compiler or another required tool is missing, rather than silently skipping required checks. Test runs must not download reference sources or dependencies.

As an additional validation, compile the Go-generated SQLite parser in the corresponding SQLite source tree and run relevant SQLite tests if the build environment permits.

Successful compilation or runtime testing is supplementary evidence, not a substitute for the golden-file acceptance criterion.

7. Documentation

Maintain docs/PORTING.md as a running engineering log.

For each milestone, record:

* C functions translated
* Go implementation files
* Tests added
* Compatibility results
* Intentional deviations
* Unresolved issues
* Next recommended task

Maintain docs/COMPATIBILITY.md describing the supported SQLite revision, command-line behavior, tested grammars, and any remaining incompatibilities.

Inventory the command-line options and grammar directives supported by the pinned C implementation. For each, identify whether it is supported, verified by differential tests, or explicitly excluded from the compatibility claim. Cover supported behavior with differential tests; matching SQLite's parse.y alone is not evidence of general-purpose Lemon compatibility.

Document which functionality has been implemented but not yet verified against the reference.

Do not claim full compatibility based only on unit tests or a successful build.

8. Definition of Done

The project is complete when:

1. The Go executable builds without cgo or external Go dependencies.
2. It processes the pinned SQLite parse.y using the reference options and template.
3. The generated parse.c passes diff -bw against the C-generated golden file with no unexplained differences.
4. Other generated artifacts match under the applicable comparison rules.
5. Generation is deterministic across repeated runs.
6. The test suite passes.
7. The reference revision and reproduction instructions are documented.
8. No C code is required to execute the finished Go implementation.

Byte-for-byte identical output is desirable but not required if the whitespace-insensitive acceptance test passes and every byte-level difference has been reviewed and documented as a behavior-preserving formatting difference.

9. Working Instructions for the Coding Agent

Work autonomously within the authorized milestones, but make small, reviewable commits. The initial authorization covers Milestone A only; the stopping instruction in Section 10 takes precedence over the broader milestone plan.

Use commit messages identifying the translated subsystem or compatibility milestone.

Do not redesign completed subsystems without evidence that the translation is incorrect.

When a test fails, investigate the earliest observable divergence between the C and Go implementations rather than patching the final generated output.

Do not hardcode SQLite’s grammar, symbol numbers, state tables, or expected generated output.

The implementation must remain a general-purpose Lemon parser generator, even though SQLite’s grammar is the principal acceptance fixture.

If a behavior is unclear, consult the pinned original C implementation rather than guessing.

Keep the repository buildable and tests passing at each completed milestone.

10. First Task

Begin with Milestone A only.

1. Inspect the repository and existing project conventions.
2. Select and pin an official SQLite source revision.
3. Retrieve the matching lemon.c, lempar.c, and parse.y.
4. Determine and reproduce SQLite’s parser-generation command.
5. Compile the C reference executable.
6. Generate and verify deterministic golden files.
7. Create docs/REFERENCE.md and an initial docs/ARCHITECTURE.md.
8. Implement the reference build and comparison scripts.
9. Commit the baseline.

Stop after Milestone A and await authorization for Milestone B. A reproducible reference baseline is necessary but does not by itself authorize translation.

At the end of Milestone A, report the source revision, exact command used, generated artifact hashes, baseline test results, and proposed next translation step.
