# Use a custom parser template

Start from the pinned [`lempar.c`](../../reference/sqlite/tool/lempar.c), not an
arbitrary C file. Lemon fills its ordered `%%` insertion points and rewrites
eligible `Parse` name prefixes. Keep those insertion points and generated macro
contracts intact. See [template responsibilities](../explanation/parser-templates.md).

With a built `lemon` on PATH, run from the repository root:

```sh
mkdir -p build
cp reference/sqlite/tool/lempar.c build/custom.c
lemon -Tbuild/custom.c -dbuild docs/tutorials/first-parser/calc.y
test -s build/calc.c
```

`-TFILE` selects exactly that file. A nonexistent explicit path is an error; it
does **not** fall back to the embedded template. After editing a template,
regenerate and compile/run your parser's own tests. Successful generation alone
does not validate the template's C code.

## Select a template without `-T`

For a grammar-local override, use the grammar's basename with `.lt`:

```sh
cp docs/tutorials/first-parser/calc.y build/calc.y
cp reference/sqlite/tool/lempar.c build/calc.lt
lemon build/calc.y
```

For a working-directory override instead, run in a directory containing
`lempar.c` and no adjacent `calc.lt`:

```sh
mkdir -p build/local-template
cp docs/tutorials/first-parser/calc.y build/local-template/calc.y
cp reference/sqlite/tool/lempar.c build/local-template/lempar.c
(cd build/local-template && lemon calc.y)
```

Lookup checks explicit `-T`, grammar-adjacent `.lt`, then working-directory
`lempar.c`. Next it checks beside the executable when its invocation name has
a slash; otherwise it searches PATH for `lempar.c`. If the final candidate is
absent, this Go port uses its embedded pinned template. Read errors on a
selected external file are not hidden. See the [exact lookup rules](../reference/output-files.md#parser-template-selection)
for filesystem edge cases.

Remove only your own override files to return to the default. A stray `.lt` or
working-directory `lempar.c` can explain why an edited default seems to have no
effect. Use explicit `-T` when reproducibility requires one specific external file.
