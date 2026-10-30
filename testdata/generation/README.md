# Complete C generation fixtures — Milestones E and F

`cases.tsv` drives both `TestGenerationOracle` and `scripts/test-generation.sh`.
Fields are name, operation, input relative to this directory (or `sqlite` or
`missing`), and optional whitespace-separated argv words. Sources are copied to
`grammar.y`; `baseline` instead uses `parse.y` and compares unchanged Milestone A
artifacts. `generate`/`baseline` use current-directory pinned `lempar.c`;
`adjacent` uses `grammar.lt`; `executable` uses `./bin/lemon` and `bin/lempar.c`.
All runs have an empty `out` directory; `missing` directories are never created.

`golden/NAME.sha256` records **only pristine C output**, as sorted GNU sha256sum
lines with relative paths. It includes stdout, stderr, exit status, every
generated artifact (even on failure), and no input/template/binary files.
Compact hashes avoid duplicating megabytes of SQLite output per configuration.
Go tests generate text and check the same inventories/hashes offline; they do
not execute C or download anything. Header timestamp preservation and stale
header replacement are checked through the command API.

Explicit full-file checks:

```sh
scripts/test-generation.sh
scripts/test-generation.sh --reference-only
scripts/test-generation.sh --update-goldens
```

The default builds pristine C and Go, verifies C's saved hashes, then compares
all actual outputs byte for byte and under `diff -bw`, without normalization.
Reference-only independently repeats C and compares actual files, without Go.
Updating writes C-derived hashes and independently repeats C; it does not derive
expectations from Go. All modes check the unchanged SQLite baseline goldens.
Missing required tools fail explicitly; all failures retain full inputs,
outputs, binaries, manifest, and environment information with a reproduction
command. Successful work directories are deleted.

`actions.y` distinguishes direct/shared/temporary LHS writes, `@` codes,
overwrite comments, datatype slots, shared/specific/default destructors, hooks,
context/argument/allocator macros, fallback and wildcard. `actions-errors.y`
exercises emission diagnostics and generated files on errors. Existing earlier
fixtures supply conflicts, nullable states, compounds, malformed grammars,
no-code chains, empty values/actions, directives, and SQLite variants.
`runtime.y` compiles and runs in the explicit script: asymmetric arithmetic
`7 - 2 * 3` must produce value 1, one acceptance, and zero errors.

Milestone F adds `runtime-recovery.y` (fallback, wildcard, error recovery,
continuing after an error, EOF failure, and token/nonterminal destruction) and
`runtime-stack.y` (400 nested pairs on a growable stack, plus a fixed eight-entry
stack that overflows, destroys seven stacked values, and accepts a subsequent
parse). Both C- and Go-generated parsers run with assertions enabled and with
`-DNDEBUG`. Recovery and dynamic-stack fixtures also run with `-c`.
`fixtures/CASE.expected` contains the independently checked, C-confirmed runtime
output for each case; runtime failures fail the explicit script. Ordinary Go
tests check only generated artifact hashes and never compile or execute C.

The compressed and uncompressed recovery cases intentionally have different
expectations. Without compression, reductions wait for an explicit FOLLOW
lookahead. BAD after `WILD OTHER SEMI` discards OTHER's value 11 instead of
executing the pending wildcard statement action; BAD/EOF after ID destroys the
terminal instead of a reduced `value`. Compression makes those reductions run
before the error. This is observed in pristine C, not a permitted C/Go difference.
These runtime checks supplement, and never replace, exact artifact comparisons.
The separate SQLite integration command is documented in
[REFERENCE.md](../../docs/REFERENCE.md).
