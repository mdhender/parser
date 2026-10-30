# Troubleshoot parser generation

Use the symptom that matches your failure. Commands assume a built `lemon` on
PATH, a POSIX shell, and the repository root as working directory unless noted.
Always check generation status before using outputs: files can exist on failure.

## “Can't find the parser driver template file”

An explicit `-T` path must exist and be readable:

```sh
lemon -Tmissing-template.c docs/tutorials/first-parser/calc.y
```

This intentionally returns nonzero and identifies `missing-template.c`. Correct
the path or omit `-T` to use default lookup and the embedded fallback. “Can't open
the template file” means a selected file could not be read; check that it is a
regular readable file. The fallback does not repair an invalid external template.
See [custom templates](custom-template.md).

## “Can't open file” for an output

`-d` does not create directories. Create a writable destination and attach its name:

```sh
mkdir -p build
lemon -dbuild docs/tutorials/first-parser/calc.y
```

Expected: zero exit status and `build/calc.c`, `.h`, `.out`. Check permissions,
the grammar path, and available storage if this still fails. `-d build` is not
equivalent: string option arguments must be attached. For “Exactly one filename
argument is required”, check for such accidentally separated arguments. See
[argument syntax](../reference/cli.md).

## A grammar diagnostic names a source line

Fix the first reported error and regenerate: later messages may be consequences.
Check lowercase rule LHS names, uppercase tokens, `::=`, final periods, balanced
braces, and aliases used in actions. Compare with [grammar syntax](../reference/grammar.md)
and the [declaration inventory](../reference/directives.md).

Inspect which conditional branches Lemon sees without generating a parser:

```sh
lemon -E -DDEMO docs/tutorials/first-parser/calc.y
```

Expected: preprocessed grammar on stdout, no newly generated parser files. Use
your own macro names with your own grammar. Lemon conditionals are not C
preprocessor expressions; directives must begin at column one.

## “parsing conflicts” or “rule can not be reduced”

Regenerate with a state report and precedence-resolved actions visible:

```sh
mkdir -p build
lemon -p -s -dbuild docs/tutorials/first-parser/calc.y
```

The supplied example reports zero conflicts. For your grammar, inspect the
affected states and rules in its `.out` file. Remove `-q` if you need that report.
For a deliberate operator grammar, declare associativity/precedence or add an
explicit rule precedence mark. Otherwise rewrite the ambiguous rules. Do not
add precedence merely to silence an unexplained conflict. Unresolved conflicts
return nonzero even when C output exists. See [how conflicts arise](../explanation/how-lemon-works.md).

## The generated C does not compile or link

Check the error's grammar source line (default `#line` directives point back to
it), required includes, semantic types, and any custom template modifications.
The token header alone does not declare the parser API. Do not separately compile
the generated C if your driver already includes it: that duplicates definitions.
The [tutorial](../tutorials/first-parser.md) demonstrates one complete arrangement.

SQLite's `parse.y` requires SQLite's own headers, types, actions, and build flags;
it cannot be linked as a generic standalone parser. Maintainers can use the
[pinned SQLite integration procedure](../../testdata/sqlite/README.md).

## Output differs from another Lemon installation

Compare generator revision, selected template, macro list/order, relative input
paths, working directory, and flags before comparing artifacts. `-x` is not a
SQLite revision identifier. Avoid `-r`: the pinned implementation skips table
initialization as well as state sorting, producing zero retained states. This
port preserves that behavior. The [compatibility reference](../reference/compatibility.md)
states the tested boundary and links to exact comparison evidence.
