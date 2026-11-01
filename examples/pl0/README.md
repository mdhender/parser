# PL/0: three Go parsers, one interpreter

A larger example inspired by Niklaus Wirth's PL/0 language in
*Algorithms + Data Structures = Programs*. This is a new implementation,
not a transcription of Wirth's compiler or its stack machine.

- [`lalr/grammar.y`](lalr/grammar.y): `glcc` LALR(1), with Go AST-building actions.
- [`ll1/grammar.y`](ll1/grammar.y): `glime` LL(1), also with Go actions,
  shared by the stack and recursive-descent frontends.
- [`recursive/`](recursive/): the recursive-descent parser generated from that
  same LL(1) grammar in a separate package.
- [`lang/`](lang/): **one** lexer, AST definition, name checker, and interpreter.
- [`programs/`](programs/): squares, nested lexical scopes, and recursive factorial.

All three frontends return `*lang.Block`. The interpreter has no dependency on a
parser. The LL(1) frontends' expression tails are folded into the same
left-associative trees as the LALR frontend; they never enter the shared AST.
Numeric token IDs belong to each generated parser, not to the shared lexer.

Learn with the [glcc tutorial](../../docs/tutorials/pl0-glcc.md),
[glime stack tutorial](../../docs/tutorials/pl0-glime.md), or
[recursive tutorial](../../docs/tutorials/pl0-recursive.md), or go straight to
[Run PL/0 with any Go parser frontend](../../docs/how-to/run-pl0.md).
The [PL/0 reference](../../docs/reference/pl0.md) specifies this example's
language, extensions, APIs, and limits. Generated parsers are checked in so the
example builds immediately; regeneration is tested against exact saved bytes.

## Frontend selection and regeneration

The CLI retains `glcc` as its default. `-parser glime` selects the existing
predictive stack parser; `-parser glime-recursive` selects recursive descent.
For example, from the repository root:

```sh
go run ./examples/pl0 -parser glime-recursive examples/pl0/programs/squares.pl0
go generate ./examples/pl0/...
```

The generation directives run these commands in their respective directories:

| Directory | Command |
| --- | --- |
| `lalr` | `go run ../../../cmd/glcc -q -package lalr grammar.y` |
| `ll1` | `go run ../../../cmd/glime -package ll1 generate grammar.y` |
| `recursive` | `go run ../../../cmd/glime -strategy=recursive -package recursive -d . generate ../ll1/grammar.y` |

Each directory owns its generated `grammar.go`. The two LL(1) packages keep
their token constants and generated `ParseTokens` APIs separate without copying
the grammar or changing its actions. Each exports `Parse(string)
(*lang.Block, error)` for use with the shared interpreter.
