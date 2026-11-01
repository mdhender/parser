# Run PL/0 with either Go parser

Use the same interpreter with either the `glcc` LALR(1) parser or the `glime`
LL(1) parser. You need a repository checkout and the Go version in
[`go.mod`](../../go.mod); no C compiler or template installation is required.
Run these commands from the repository root.

## Run a supplied program

```sh
go run ./examples/pl0 -parser glcc examples/pl0/programs/squares.pl0
go run ./examples/pl0 -parser glime examples/pl0/programs/squares.pl0
```

Each command prints:

```text
1
4
9
16
25
55
```

The program computes the first five squares with a procedure, then prints their
sum. Try the other supplied programs with either `-parser` value:

| Program | Expected output, one value per line |
| --- | --- |
| `examples/pl0/programs/scopes.pl0` | `11`, `3`, `3` |
| `examples/pl0/programs/factorial.pl0` | `720` |

To run your own program, replace the filename. The default parser is `glcc`.
Place flags before the filename. Errors go to stderr with filename, line, and
column where available; a failed invocation returns a nonzero exit status.
See the [language reference](../reference/pl0.md) for supported syntax.

## Regenerate after editing a grammar

The checked-in generated parsers mean normal builds and runs do not regenerate
anything. After changing either grammar, regenerate both:

```sh
go generate ./examples/pl0/...
go test ./examples/pl0/...
```

The directives invoke the repository's `glcc` and `glime` commands. The
equivalent explicit commands, from the repository root, are:

```sh
go run ./cmd/glcc -q -package lalr examples/pl0/lalr/grammar.y
go run ./cmd/glime -package ll1 generate examples/pl0/ll1/grammar.y
```

Inspect the predictive grammar without generating files:

```sh
go run ./cmd/glime check examples/pl0/ll1/grammar.y
go run ./cmd/glime explain examples/pl0/ll1/grammar.y
```

The example tests compare ASTs from both frontends, check independently expected
results and expression-tree shapes, reject malformed programs, and require
fresh generation to reproduce both checked-in parsers exactly.

For smaller starting points, use the [Go calculator](../tutorials/go-parser.md)
or [LL(1) calculator](../tutorials/ll1-parser.md) tutorial.
