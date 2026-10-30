# Your first C parser

Build a parser that adds two integers. You will give it tokens directly, so you
do not need to write a lexer yet. The generator is written in Go; the parser
you generate and run is C.

## Before you start

Use a checkout of this repository, Go **1.27.3** (the version in
[`go.mod`](../../go.mod)), a C compiler available as `cc`, and a POSIX shell.
The commands below run from the repository root on Linux. No separate template
installation is needed: the Go executable embeds the pinned `lempar.c`.
See [installation](../how-to/install.md) if you need to build the generator first.

## Read the grammar

The complete example is [`calc.y`](first-parser/calc.y):

```lemon
%include {
#include <stdio.h>
#include <stdlib.h>
}
%token_type {int}
%syntax_error { fprintf(stderr, "Syntax error\n"); exit(1); }
%parse_failure { fprintf(stderr, "Parse failed\n"); exit(1); }

input ::= sum.
sum ::= INTEGER(A) PLUS INTEGER(B). { printf("Result: %d\n", A + B); }
```

Uppercase names are tokens supplied by the caller. Lowercase names are grammar
symbols assembled by the parser. Each production ends with a period. Here,
`input` is the start symbol because its rule comes first.

`%token_type {int}` makes each token's value a C integer. The aliases `A` and
`B` name the two integer values inside the reduction action. When the `sum`
rule reduces, its C action adds those values and prints the result.

## Read the caller

The complete caller is [`main.c`](first-parser/main.c):

```c
#include "calc.c"
#include "calc.h"

int main(void) {
    void *parser = ParseAlloc(malloc);
    if (parser == NULL) return 1;
    Parse(parser, INTEGER, 3);
    Parse(parser, PLUS, 0);
    Parse(parser, INTEGER, 4);
    Parse(parser, 0, 0);
    ParseFree(parser, free);
    return 0;
}
```

`ParseAlloc` creates a parser instance. Each `Parse` call supplies a token code
and value; the value on `PLUS` is unused. Token code **0** signals end of input,
allowing the parser to finish. `ParseFree` releases the parser.

For this small example the caller includes the generated C file directly, so
the API definitions are visible without a separate declarations header. Compile
only `main.c`, not `main.c` and `calc.c` together. `calc.h` supplies token constants;
it does not declare the parser API. In a larger program you can compile the
generated C separately and provide matching API declarations in your own header.

## Generate, compile, and run

Run this block from the repository root. It builds the generator and copies the
two supplied files into a fresh temporary directory, then removes that directory
when the shell exits. It does not modify your checkout.

```sh
set -eu
repo=$PWD
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
CGO_ENABLED=0 go build -o "$work/lemon" ./cmd/lemon
cp "$repo/docs/tutorials/first-parser/calc.y" "$work/calc.y"
cp "$repo/docs/tutorials/first-parser/main.c" "$work/main.c"
cd "$work"
./lemon calc.y
test -s calc.c
test -s calc.h
test -s calc.out
cc -o calculator main.c
./calculator
```

The output is exactly:

```text
Result: 7
```

You generated `calc.c` (the parser), `calc.h` (token constants), and `calc.out`
(a state report). The embedded template provided the parser's stack and runtime;
your grammar provided the rules and action. The C executable does not call the
Go generator at runtime.

You now have a working push parser. Next, [generate a parser from your own
grammar](../how-to/generate-parser.md), or look up [grammar syntax](../reference/grammar.md).
For the conceptual model, read [how Lemon works](../explanation/how-lemon-works.md).
