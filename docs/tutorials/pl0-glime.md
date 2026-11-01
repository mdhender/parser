# Build a PL/0 AST with glime

We will check and generate the PL/0 LL(1) parser, build a caller that inspects
its AST, and run that tree with the shared interpreter. Our program calls a
procedure twice and prints **10**. This tutorial stands on its own; completing
[the glcc tutorial](pl0-glcc.md) first is not required.

## Before you start

Use this repository checkout, the Go version in [`go.mod`](../../go.mod), and
a POSIX shell. No C compiler, cgo, or template installation is needed. Start at
the repository root and keep the same shell open throughout.

Run the supplied squares program first:

```sh
set -eu
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
go run ./examples/pl0 -parser glime examples/pl0/programs/squares.pl0
```

You should see `1`, `4`, `9`, `16`, `25`, and `55`, each on its own line.
The last value is the sum. This uses the existing LL(1) parser and the one
shared interpreter; we will now regenerate that parser and write our own caller.

## Check the predictive grammar

Open the supplied [grammar](../../examples/pl0/ll1/grammar.y). Find `expr`,
`term`, `sum_tail`, and `product_tail`. Unlike the LALR grammar, these
productions have no left recursion. A tail chooses between an operator and
another term, or an empty production when the expression ends.

Build glime and check the entire grammar:

```sh
go build -o "$work/glime" ./cmd/glime
"$work/glime" check examples/pl0/ll1/grammar.y
"$work/glime" explain examples/pl0/ll1/grammar.y >"$work/predictions.txt"
test -s "$work/predictions.txt"
```

The check succeeds silently. Open `$work/predictions.txt` to see the computed
nullable, FIRST, FOLLOW, SELECT, and prediction information. Find `sum_tail`:
its PLUS/MINUS productions consume an operator; its empty alternative is
predicted by tokens that can follow the expression. There are no conflicts.
Neither `check` nor `explain` writes a parser file.

## Generate the stack parser

```sh
"$work/glime" -strategy=stack -package ll1 generate examples/pl0/ll1/grammar.y
test -s examples/pl0/ll1/grammar.go
```

Generation succeeds silently. The resulting `grammar.go` contains predictive
tables, typed semantic frames, and the Go runtime. It is already checked in;
regenerating should not change it. Edit the grammar, not generated Go.

## Write a PL/0 program

Create this program in our temporary directory:

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

The procedure changes the enclosing variable `x` twice. The final `write`, an
example-language extension, will print 4 + 3 + 3, or 10.

## Inspect and execute the tree

Write a complete caller:

```sh
cat >"$work/main.go" <<'GO'
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/mdhender/parser/examples/pl0/lang"
    "github.com/mdhender/parser/examples/pl0/ll1"
)

func main() {
    source, err := os.ReadFile(os.Args[1])
    if err != nil { log.Fatal(err) }
    tree, err := ll1.Parse(string(source))
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

Notice that only the frontend import and parse call differ from the LALR
caller. Both return `*lang.Block`, and **both call `lang.Execute`**. There is
no second interpreter and no LL(1)-specific node in the shared AST.

## Trace a deferred AST action

Find this production in the grammar:

```lemon
sum_tail(A) ::= MINUS(O) term(B) sum_tail(T). { A = append([]operation{{token:O,right:B}},T...) }
```

It collects operator/operand pairs without executing PL/0 arithmetic. Open
[`ll1/parse.go`](../../examples/pl0/ll1/parse.go) and find `fold`. For
`19-6-2`, it first constructs the node `19-6`, then uses that node as the left
child of the subtraction of 2. The tree is `(19-6)-2`, not `19-(6-2)`.

Return to the `expr` production: `fold` runs only after its term and tail have
completed. The generated runtime's completion markers enforce that order, even
inside nested parentheses. Only ordinary expression nodes reach the interpreter.

Finally, find the adapter's call to `ParseTokens`. It passes the whole token
slice, omitting the shared lexer's explicit EOF token: glime supplies EOF
implicitly. This differs from the glcc push API, but the public example wrapper
keeps the same `Parse(source)` signature.

You have generated a predictive Go parser that builds the same shared AST as
the LALR parser. Continue with the [PL/0 reference](../reference/pl0.md) for the
language contract, [glime reference](../reference/glime.md) for supported
features, or [the glcc tutorial](pl0-glcc.md) to compare the two frontends.
