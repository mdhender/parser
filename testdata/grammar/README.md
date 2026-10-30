# Grammar-processing oracle fixtures

These are Milestone B intermediate results, not generated-parser acceptance
artifacts. The instrumentation includes the pristine pinned `lemon.c`, renaming
its entry point to `lemon_main`; it has its own entry point and stops immediately
after `FindRulePrecedences`. Final parser comparisons must use the pristine C
executable instead.

## Commands

```sh
scripts/test-grammar.sh                    # build cmd/lemon; C goldens + C/Go
scripts/test-grammar.sh --reference-only   # C goldens + repeated C execution
scripts/test-grammar.sh --update-goldens   # explicitly regenerate, then repeat
cc -o /tmp/lemon-snapshot reference/grammar-snapshot.c
/tmp/lemon-snapshot -DALPHA -UBETA -l grammar.y
```

The script requires Bash, a C compiler (`CC`, default `cc`), and ordinary Unix
comparison/file tools. Differential mode additionally requires the installed Go
toolchain from `go.mod`. All modes are offline; missing tools fail, never skip.
`CC` must name a single executable, not a compiler-plus-options command string.
`LC_ALL=C` and `TZ=UTC` are fixed. Go downloads and toolchain downloads are disabled.

`cases.tsv` has four tab-separated columns: case name, operation (`snapshot`,
`reprint`, `preprocess`), fixture basename (or `sqlite`/`missing`), and optional
space-separated options. Options cannot contain whitespace. The SQLite input is
`reference/sqlite/src/parse.y`; no second copy is checked in here.

Each invocation gets a new isolated directory and the same relative input name
`grammar.y`, with executable argv[0] `./lemon`. The C driver accepts only upstream
`-DNAME[=VALUE]`, `-UNAME`, and `-l` options, parsed by the included `OptInit`.
Go receives those options plus `--grammar-snapshot`. For reprint/preprocess,
both executables receive upstream `-g`/`-E`, and C uses the **uninstrumented**
executable. The invalid-option case also uses the pristine executable, since the
snapshot driver's intentionally smaller usage list is not an upstream CLI oracle.

Reprint/preprocess cases also exercise CLI flag prefixes and trailing text,
plus/minus flag overrides, generation-only and ignored options, bare `D=ALPHA`
and `U=ALPHA` callbacks, short/long diagnostic caret placement, separated template
arguments, and version handling before filename-count validation. Operations
prepend `-g` or `-E` before manifest options; later options can override that mode.
All successful invocations remain reprint/preprocess-only: `+g` is followed by
`-g`, or used with `-E`, so it cannot accidentally invoke parser generation.
The harness always appends `grammar.y`; zero-filename CLI behavior is not covered.

`preprocess-quirks.y` covers whole-expression short-circuiting rather than C
operator precedence, syntax skipped by short-circuiting or an excluded nesting
level, indented directives preserved inside code, and a missing active `%endif`
accepted upstream. `destructor-repeat.y` covers appended per-symbol destructor
fragments, first-fragment line metadata, and raw UTF-8 declaration/type/action
bytes, both with and without line directives. Non-ASCII identifier bytes are a
separate malformed-input case under the fixed C locale.

Every `golden/CASE/` contains `stdout.txt`, `stderr.txt`, and `exit-status.txt`
(decimal shell status followed by newline). All other generated files are also
compared if present. Both `cmp` and `diff -bw` are required to pass; whitespace-only
differences are failures, not automatically accepted. On failure, the script
prints a reproduction command and retains the complete temporary directory,
including C and Go binaries, inputs, stdout, stderr, and status. Successful runs
remove their temporary directory. Goldens can be read by ordinary Go tests with
no C compiler and no network.

## Snapshot TSV contract

Every record ends in newline. All string fields are lowercase hexadecimal raw
bytes (including UTF-8, embedded tabs/newlines, and C `#line` directives). A NULL
string or absent symbol is `-`; an allocated empty string is an empty field.
Keywords and integer fields are not hex encoded. Empty lists are empty fields;
list entries are comma separated.

Records, in order:

* `grammar`: nsymbol, nterminal, nruleWithAction, has_fallback, wildcard name,
  original startRule index (or -1).
* Eighteen `decl` records: keyword, nullable value. Keyword order is name,
  include, code, token_destructor, default_destructor, token_prefix, syntax_error,
  parse_accept, parse_failure, stack_overflow, extra_argument, extra_context,
  token_type, default_type, realloc, free, stack_size, start_symbol.
* `symbol`: name, index, type integer, useCnt, prec, assoc integer, fallback name,
  nullable destructor, destLineno, nullable datatype, bContent, constituent names,
  original rule indices in nextlhs order. The sorted symbol array includes
  `{default}` and named multiterminals beyond `nsymbol`; anonymous inline
  compounds occur only in RHS records.
* `rule`: original index, action-first iRule, lhs name, nullable lhsalias,
  ruleline, line, nullable code, noCode integer, neverReduce integer, precsym name,
  RHS count. Rules are in C's action-first order, not original input order.
* Each rule is immediately followed by its `rhs` records: name, type integer,
  nullable alias, constituent names.
* `define`: name, used integer, in the sorted define order established before
  preprocessing (duplicates remain distinct). These records come last.

The driver does not run FIRST-set calculation, state construction, code
generation, or runtime checks. Header `startRule` is the original first rule,
not a newly selected rule for `%start_symbol`; that later work belongs to state
construction. Malformed inputs return the upstream parser's error count before
any snapshot; preprocessing failures may terminate directly in upstream code.
