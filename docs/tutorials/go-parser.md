# Your first Go parser

Generate a calculator that evaluates `2 + 3 * (4 - 1)` to **11**. You will
supply tokens directly; writing a lexer is a separate task. Both the actions
and the generated parser are Go, unlike the [C tutorial](first-parser.md).

## Before you start

Use a checkout of this repository, Go **1.27.3** (see
[`go.mod`](../../go.mod)), and a POSIX shell. No C compiler, cgo, external Go
dependencies, or template installation is needed. Run the following from the
repository root, keeping the same shell open throughout:

```sh
set -eu
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
CGO_ENABLED=0 go build -o "$work/glcc" ./cmd/glcc
```

`glemon` is an equivalent command: you can build `./cmd/glemon` instead.

## Write the grammar

Save this complete grammar as `$work/calc.y` (expand `$work` to the temporary
directory printed by `printf '%s\n' "$work"` if using an editor):

```lemon
%name Calc
%token_prefix TK_
%token_type {int}
%default_type {int}
%extra_argument {out *int}

%left PLUS MINUS.
%left TIMES.

%syntax_error { panic("syntax error") }
%parse_failure { panic("parse failed") }
%stack_overflow { panic("parser stack overflow") }

input ::= expr(A). { *out = A }
expr(A) ::= expr(B) PLUS expr(C). { A = B + C }
expr(A) ::= expr(B) MINUS expr(C). { A = B - C }
expr(A) ::= expr(B) TIMES expr(C). { A = B * C }
expr(A) ::= LPAREN expr(B) RPAREN. { A = B }
expr(A) ::= INTEGER(B). { A = B }
```

Uppercase names are tokens; lowercase names are nonterminals. A production ends
in a period, followed by an optional Go action in braces. The first rule makes
`input` the start symbol. Later precedence declarations bind more tightly, so
`TIMES` binds more tightly than `PLUS` and `MINUS`. Parentheses group expressions
through an explicit rule.

Aliases are **typed Go locals**. In `expr(A) ::= expr(B) PLUS expr(C)`, `A`
starts at the zero value and the action assigns `B + C` to it. The separate
left-hand alias must differ from every right-hand alias. Even a forwarding rule
needs `A = B`: there is no implicit copy from the right-hand side.

`%extra_argument {out *int}` adds an `out` parameter to each parser call. The
start rule stores the answer through that pointer; it does not merely rebind
the local variable. `%name Calc` names the parser type and constructor, while
`%token_prefix TK_` names constants such as `TK_INTEGER`.

## Write the caller

Save this as `$work/main.go`:

```go
package main

import "fmt"

func main() {
	p := NewCalc()
	defer p.Close()
	var result int
	// Supply 2 + 3 * (4 - 1), one token and semantic value at a time.
	for _, token := range []struct{ kind, value int }{
		{TK_INTEGER, 2},
		{TK_PLUS, 0},
		{TK_INTEGER, 3},
		{TK_TIMES, 0},
		{TK_LPAREN, 0},
		{TK_INTEGER, 4},
		{TK_MINUS, 0},
		{TK_INTEGER, 1},
		{TK_RPAREN, 0},
	} {
		p.Parse(token.kind, token.value, &result)
	}
	p.Parse(0, 0, &result) // End of input: token 0 and int's zero value.
	fmt.Printf("Result: %d\n", result)
}
```

`NewCalc()` constructs a parser. `Parse` pushes one token; punctuation values
are unused here. Token code **0** means end of input, not an ordinary token:
feed it so pending reductions and acceptance can finish. `Close()` destroys
any values still on the stack, including when you abandon an incomplete parse.

## Generate and run

```sh
"$work/glcc" -package main -d "$work" "$work/calc.y"
test -s "$work/calc.go"
test -s "$work/calc.out"
go run "$work/main.go" "$work/calc.go"
```

The output is exactly:

```text
Result: 11
```

`calc.go` contains both token constants and the parser runtime; there is no
separate token header. `calc.out` is the state report, useful for diagnosing
conflicts. `-package main` lets the generated file build alongside this caller;
the default package is `parser`. Go flags must appear **before** the grammar
filename. To omit the report, add `-q` before that filename.

The parser uses only the standard library and does not invoke the generator at
runtime. For a reusable package, generate with `-package parser` into your
module's parser directory and import it from a separate caller. Continue with
the [Go generator/API reference](../reference/go-parser.md) for heterogeneous
values, context parameters, hooks, destructors, and templates, or the shared
[grammar syntax](../reference/grammar.md) for productions and precedence.
