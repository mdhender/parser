# Complete C generation fixtures — Milestone E

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
`7 - 2 * 3` must produce value 1, one acceptance, and zero errors. This smoke
test supplements exact artifact comparison; it is not SQLite runtime testing
or final Milestone F compatibility certification.
