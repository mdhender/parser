# Lemon parser generator in Go

A standalone Go port of SQLite's Lemon parser generator. **It generates C
parsers, not Go parsers.** Give it a Lemon grammar and it emits a C parser,
token constants, and a state report. No cgo, C compiler, or external template is
needed to run the generator; a C compiler is needed to build your generated parser.

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

## Documentation

| What do you need? | Start here |
| --- | --- |
| Learn by building a working parser | [First-parser tutorial](docs/tutorials/first-parser.md) |
| Solve a specific task | [Install](docs/how-to/install.md), [generate C](docs/how-to/generate-parser.md), [customize the template](docs/how-to/custom-template.md), [troubleshoot](docs/how-to/troubleshoot.md) |
| Look up syntax and behavior | [CLI](docs/reference/cli.md), [grammar](docs/reference/grammar.md), [directives](docs/reference/directives.md), [outputs](docs/reference/output-files.md) |
| Understand the design | [How Lemon works](docs/explanation/how-lemon-works.md), [templates](docs/explanation/parser-templates.md), [Lemon vs yacc/Bison](docs/explanation/lemon-vs-yacc.md) |

See the [documentation index](docs/README.md) for all four Diátaxis categories.

## Compatibility and development

Generated artifacts match the pinned **SQLite 3.50.4** C reference byte for byte
for the recorded test matrix. This is not a claim of exhaustive compatibility
with every Lemon release, grammar, template, or platform. The Go port adds
an embedded default template and optional internal snapshot modes.
See the [compatibility summary](docs/reference/compatibility.md) for boundaries.

For offline Go tests, explicit C differential checks, provenance, architecture,
and the engineering log, use the [development documentation](docs/development/README.md).
Release history is in [CHANGELOG.md](CHANGELOG.md).
