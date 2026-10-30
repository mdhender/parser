# Generate C from an existing grammar

You need a built `lemon` executable and a Lemon grammar. This guide uses the
repository's small example; replace its path with your own grammar afterward.
Run the following from the repository root with `lemon` on PATH:

```sh
mkdir -p build
lemon -dbuild docs/tutorials/first-parser/calc.y
test -s build/calc.c
test -s build/calc.h
test -s build/calc.out
```

`-d` takes an **attached** directory name: `-dbuild`, not `-d build`.
Create the directory first. Without `-d`, outputs are written beside the grammar,
not necessarily in your working directory. Existing generated files may be
overwritten; use a dedicated directory and check the command's exit status.

The default embedded template needs no installation. To choose an external
template, follow [use a custom template](custom-template.md).

## Choose the artifacts you need

For source and token header without the state report:

```sh
lemon -q -dbuild docs/tutorials/first-parser/calc.y
```

`-q` prevents writing a report; it does not delete an old `calc.out`. To add
grammar metadata in SQL form and print statistics:

```sh
lemon -S -s -dbuild docs/tutorials/first-parser/calc.y
test -s build/calc.sql
```

A successful command returns zero. Nonzero status can leave partial artifacts;
do not compile them as though generation succeeded. Consult the
[troubleshooting guide](troubleshoot.md) if diagnostics mention conflicts or files.

## Integrate with C

Supply tokens and their values from your lexer, finish with token code 0, and
release the parser instance. The header contains token constants, not parser API
prototypes. The [tutorial](../tutorials/first-parser.md) supplies a complete caller
and compile command. Your grammar's C actions and includes determine any other
headers, libraries, or compiler flags needed; SQLite's full grammar is not a
standalone example.

See [CLI options](../reference/cli.md) and [output files](../reference/output-files.md)
for the complete controls and generated API.
