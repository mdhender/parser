# Lemon parser generator in Go

A standalone Go port of SQLite's Lemon parser generator. Choose `lemon` to
generate **C parsers**, or `glcc` / `glemon` to generate **Go parsers** from
Lemon grammars with Go actions. The Go commands are equivalent thin wrappers
around the same generator entry point, `RunGo`; `lemon` has no language flag
and retains its C behavior.

For LL(1) grammars, the separate **`glime`** command checks/explains predictions
and generates standalone Go parsers using a predictive stack and deferred typed
actions. See the [LL(1) tutorial](docs/tutorials/ll1-parser.md) and
[glime reference](docs/reference/glime.md). It does not change the LALR commands.

The generators need no cgo, C compiler, external Go dependencies, or separately
installed default template. Building a generated C parser needs a C compiler;
the generated Go runtime uses only the Go standard library.

## Try it

From a checkout, with Go **1.27.3**:

```sh
CGO_ENABLED=0 go build -o lemon ./cmd/lemon
mkdir -p build
./lemon -dbuild docs/tutorials/first-parser/calc.y
```

This writes `build/calc.c`, `build/calc.h`, and `build/calc.out`, using the
embedded SQLite 3.50.4 template. Follow the complete
[first-parser tutorial](docs/tutorials/first-parser.md) to compile and run it.

For a Go parser, build either command and use **Go flag syntax**, with options
before the grammar filename:

```sh
CGO_ENABLED=0 go build -o glcc ./cmd/glcc
./glcc -package parser -d build calculator.y
# Equivalent: go build -o glemon ./cmd/glemon
```

Here `calculator.y` must contain Go actions, not C actions. This writes
`build/calculator.go` and `build/calculator.out`. Follow the runnable
[Go calculator tutorial](docs/tutorials/go-parser.md), or consult the
[Go generator and runtime reference](docs/reference/go-parser.md).

For a larger example, [run PL/0 with either Go parser](docs/how-to/run-pl0.md).
Its `glcc` and `glime` grammars both build the same AST and use **one shared
interpreter**, with nested procedures, lexical scopes, loops, and recursion.
Follow the separate [glcc PL/0](docs/tutorials/pl0-glcc.md) or
[glime PL/0](docs/tutorials/pl0-glime.md) tutorial to generate and call each parser.

## Documentation

| What do you need? | Start here |
| --- | --- |
| Learn by building a working parser | [First C parser](docs/tutorials/first-parser.md), [Go calculator](docs/tutorials/go-parser.md) |
| Solve a specific task | [Install](docs/how-to/install.md), [generate C](docs/how-to/generate-parser.md), [customize the template](docs/how-to/custom-template.md), [troubleshoot](docs/how-to/troubleshoot.md) |
| Look up syntax and behavior | [CLI](docs/reference/cli.md), [Go generator/API](docs/reference/go-parser.md), [grammar](docs/reference/grammar.md), [directives](docs/reference/directives.md), [C outputs](docs/reference/output-files.md) |
| Understand the design | [How Lemon works](docs/explanation/how-lemon-works.md), [templates](docs/explanation/parser-templates.md), [Lemon vs yacc/Bison](docs/explanation/lemon-vs-yacc.md) |

See the [documentation index](docs/README.md) for all four Diátaxis categories.

## Compatibility and development

Generated C artifacts match the pinned **SQLite 3.50.4** C reference byte for byte
for the recorded test matrix. This is not a claim of exhaustive compatibility
with every Lemon release, grammar, template, or platform. The Go port adds
an embedded default template and optional internal snapshot modes.
See the [compatibility summary](docs/reference/compatibility.md) for boundaries.
That C differential evidence is not a compatibility claim for the separate Go
runtime or its templates; see the [Go-specific contract](docs/reference/go-parser.md).

For offline Go tests, explicit C differential checks, provenance, architecture,
and the engineering log, use the [development documentation](docs/development/README.md).
Release history is in [CHANGELOG.md](CHANGELOG.md).
