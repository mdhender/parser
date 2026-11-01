# Build a PL/0 AST with glcc

We will generate the PL/0 LALR(1) parser, build a small caller that inspects its
AST, and execute that tree with the shared interpreter. Our program calls a
procedure twice and prints **10**. We will use the complete supplied grammar
rather than build up the language one production at a time.

## Before you start

Use this repository checkout, the Go version in [`go.mod`](../../go.mod), and
a POSIX shell. You do not need a C compiler, cgo, or an external parser template.
Start at the repository root and keep the same shell open throughout.

First run the supplied squares program:

```sh
set -eu
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
go run ./examples/pl0 -parser glcc examples/pl0/programs/squares.pl0
```

You should see `1`, `4`, `9`, `16`, `25`, and `55`, each on its own line.
The last value is the sum. This establishes that the supplied interpreter
and parser work before we write our own caller.

## Generate the LALR parser

Open the supplied [grammar](../../examples/pl0/lalr/grammar.y). Find these rules:

```lemon
expr(A) ::= expr(B) MINUS(O) term(C). { A = &lang.Expr{Token:O,Left:B,Right:C} }
term(A) ::= term(B) TIMES(O) factor(C). { A = &lang.Expr{Token:O,Left:B,Right:C} }
```

Notice that the Go actions construct `lang.Expr` nodes: they do not subtract or
multiply PL/0 values. The left-recursive expression and term levels build
left-associative trees, with multiplication nested more tightly than subtraction.

Build the generator, then regenerate the parser in its existing package:

```sh
go build -o "$work/glcc" ./cmd/glcc
"$work/glcc" -q -package lalr examples/pl0/lalr/grammar.y
test -s examples/pl0/lalr/grammar.go
```

Generation succeeds silently. `grammar.go` now contains the numeric tables,
runtime, and typed reduction actions. `-q` omits the state report. The generated
file is already checked in, so repeating this command should not change it.
Do not edit generated Go; edit `grammar.y` and regenerate instead.

## Write a PL/0 program

Create a program in our temporary directory:

```sh
cat >"$work/demo.pl0" <<'PL0'
const step = 3;
var x;
procedure bump;
begin
  x := x + step
end;
begin
  x := 4;
  call bump;
  call bump;
  write x
end.
PL0
```

`bump` changes the enclosing variable `x`. `write` is this example's output
extension. We expect `x` to become 4 + 3 + 3, or 10.

## Inspect the AST, then interpret it

Create a complete Go caller:

```sh
cat >"$work/main.go" <<'GO'
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/mdhender/parser/examples/pl0/lalr"
    "github.com/mdhender/parser/examples/pl0/lang"
)

func main() {
    source, err := os.ReadFile(os.Args[1])
    if err != nil { log.Fatal(err) }
    tree, err := lalr.Parse(string(source))
    if err != nil { log.Fatal(err) }
    fmt.Printf("AST: %d procedure(s), %d statements\n",
        len(tree.Procedures), len(tree.Statement.Body))
    if err := lang.Execute(tree, os.Stdout); err != nil { log.Fatal(err) }
}
GO
go run "$work/main.go" "$work/demo.pl0"
```

Expected output:

```text
AST: 1 procedure(s), 4 statements
10
```

The first line comes from inspecting ordinary Go AST fields. No PL/0 statement
has executed yet. The next line comes from `lang.Execute`, the same interpreter
used by the CLI and the LL(1) frontend.

## Follow one action through the parser

In the grammar, find:

```lemon
statement(A) ::= IDENT(N) ASSIGN expr(E). { A = &lang.Statement{Kind:"assign",Token:N,Expr:E} }
```

For `x := 4`, the action stores the name token and expression node in a
`Statement`. Follow it to `block`: that action collects declarations, procedures,
and the final statement into `lang.Block`. The start rule stores that block in
the output supplied by the adapter.

Now open [`lalr/parse.go`](../../examples/pl0/lalr/parse.go). Locate the loop
calling `p.Parse`. It maps the shared lexer's symbolic tokens to this parser's
numeric IDs and feeds them one at a time. The final token has kind zero: without
that EOF, pending reductions and acceptance would not finish. The wrapper
returns the completed tree and turns syntax-hook failures into positioned errors.

You have generated a Go LALR parser and used its AST separately from execution.
Continue with [the glime version](pl0-glime.md) to construct the same tree with
LL(1) productions. For syntax and execution rules, use the
[PL/0 reference](../reference/pl0.md); for the generator API, use the
[Go parser reference](../reference/go-parser.md).
