# Build a PL/0 AST with recursive descent

We will generate a recursive-descent PL/0 parser, inspect its shared AST, and
execute a program that prints **10**. Use this checkout, Go 1.27.3, and a POSIX
shell. No C compiler is needed. Start at the repository root and keep the same
shell open throughout.

## Run the example

```sh
set -eu
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
go run ./examples/pl0 -parser glime-recursive examples/pl0/programs/squares.pl0
```

You should see `1`, `4`, `9`, `16`, `25`, and `55`, each on its own line.
This frontend uses the same lexer, AST, and interpreter as the stack and LALR
frontends. We will now regenerate it and call it directly.

## Check and generate the parser

Open [the LL(1) grammar](../../examples/pl0/ll1/grammar.y). Find `factor`:
IDENT, NUMBER, and LPAREN each select a different production. Now check the
grammar and save its predictions:

```sh
go build -o "$work/glime" ./cmd/glime
"$work/glime" -strategy=recursive check examples/pl0/ll1/grammar.y
"$work/glime" -strategy=recursive explain examples/pl0/ll1/grammar.y >"$work/predictions.txt"
test -s "$work/predictions.txt"
"$work/glime" -strategy=recursive -package recursive -d examples/pl0/recursive generate examples/pl0/ll1/grammar.y
test -s examples/pl0/recursive/grammar.go
```

Generation succeeds silently. Open `examples/pl0/recursive/grammar.go` and find
the methods named `parse` followed by a number. Each represents a nonterminal;
its switch selects productions from lookahead, then calls its children before
reducing the completed production. The nullable alternatives explicitly select
FOLLOW tokens, including zero for implicit EOF.

The grammar's `%name ParseTokens` and `%token_prefix TK_` are unchanged. Separate
`ll1` and `recursive` packages allow both generated parsers to coexist without
colliding names. Both generate a file named `grammar.go`; the output directory
keeps this command from replacing the stack parser. Regeneration is also
available through `go generate ./examples/pl0/...`.

## Create and inspect a program

Write a program that calls a procedure twice:

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
cat >"$work/main.go" <<'GO'
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/mdhender/parser/examples/pl0/lang"
    "github.com/mdhender/parser/examples/pl0/recursive"
)

func main() {
    source, err := os.ReadFile(os.Args[1])
    if err != nil { log.Fatal(err) }
    tree, err := recursive.Parse(string(source))
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

Notice that `recursive.Parse` returns `*lang.Block` and the caller invokes
`lang.Execute`, just like the other frontends. The adapter omits the lexer's
EOF token: the generated parser supplies EOF implicitly. Expression tails are
folded from the left, so `19-6-2` still builds `(19-6)-2`.

You have generated and called a recursive-descent parser without changing the
grammar or interpreter. Recursive grammar calls consume Go stack space;
4,096 nesting levels are tested, not an unlimited depth guarantee. See the
[glime reference](../reference/glime.md) for resource limits and supported
directives, or compare the [stack tutorial](pl0-glime.md).
