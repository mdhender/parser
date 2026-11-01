# PL/0: two Go parsers, one interpreter

A larger example inspired by Niklaus Wirth's PL/0 language in
*Algorithms + Data Structures = Programs*. This is a new implementation,
not a transcription of Wirth's compiler or its stack machine.

- [`lalr/grammar.y`](lalr/grammar.y): `glcc` LALR(1), with Go AST-building actions.
- [`ll1/grammar.y`](ll1/grammar.y): `glime` LL(1), also with Go actions.
- [`lang/`](lang/): **one** lexer, AST definition, name checker, and interpreter.
- [`programs/`](programs/): squares, nested lexical scopes, and recursive factorial.

Both frontends return `*lang.Block`. The interpreter has no dependency on either
parser. The LL(1) frontend's expression tails are folded into the same
left-associative trees as the LALR frontend; they never enter the shared AST.
Numeric token IDs belong to each generated parser, not to the shared lexer.

Learn with the [glcc tutorial](../../docs/tutorials/pl0-glcc.md) or
[glime tutorial](../../docs/tutorials/pl0-glime.md), or go straight to
[Run PL/0 with either Go parser](../../docs/how-to/run-pl0.md).
The [PL/0 reference](../../docs/reference/pl0.md) specifies this example's
language, extensions, APIs, and limits. Generated parsers are checked in so the
example builds immediately; regeneration is tested against exact saved bytes.
