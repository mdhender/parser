# Run PL/0 with any Go parser frontend

Use the same interpreter with the `glcc` LALR(1) parser or either `glime`
LL(1) strategy. You need a repository checkout and the Go version in
[`go.mod`](../../go.mod); no C compiler or template installation is required.
Run these commands from the repository root.

## Run a supplied program

```sh
go run ./examples/pl0 -parser glcc examples/pl0/programs/squares.pl0
go run ./examples/pl0 -parser glime examples/pl0/programs/squares.pl0
go run ./examples/pl0 -parser glime-recursive examples/pl0/programs/squares.pl0
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
sum. Try the other supplied programs with any `-parser` value:

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
anything. After changing either grammar, regenerate all three frontends:

```sh
go generate ./examples/pl0/...
go test ./examples/pl0/...
```

The directives invoke the repository's `glcc` and `glime` commands. The
equivalent explicit commands, from the repository root, are:

```sh
go run ./cmd/glcc -q -package lalr examples/pl0/lalr/grammar.y
go run ./cmd/glime -package ll1 generate examples/pl0/ll1/grammar.y
go run ./cmd/glime -strategy=recursive -package recursive -d examples/pl0/recursive generate examples/pl0/ll1/grammar.y
```

Inspect the predictive grammar without generating files:

```sh
go run ./cmd/glime check examples/pl0/ll1/grammar.y
go run ./cmd/glime explain examples/pl0/ll1/grammar.y
```

The two glime strategies share one grammar but write separate packages so their
generated parser names and token constants coexist. `-parser glime` continues
to mean stack; the default CLI frontend remains `glcc`.

The example tests compare ASTs from all three frontends, check independently expected
results and expression-tree shapes, reject malformed programs, and require
fresh generation to reproduce all checked-in parsers exactly.

For smaller starting points, use the [Go calculator](../tutorials/go-parser.md)
or [LL(1) calculator](../tutorials/ll1-parser.md) tutorial.
