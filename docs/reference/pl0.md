# PL/0 example reference

The [PL/0 example](../../examples/pl0/README.md) has three Go parser frontends
and one shared AST, name checker, and tree-walking interpreter. It is inspired
by Niklaus Wirth's PL/0, not a reproduction of his compiler, diagnostics,
bytecode, or machine limits. [Run instructions](../how-to/run-pl0.md) are separate.

## Command

```text
pl0 [-parser glcc|glime|glime-recursive] program.pl0
```

The command lives at `./examples/pl0`. The default frontend is `glcc`.
Flags precede the single input filename. `-h` succeeds; other invocation,
read, lexical, syntax, semantic, or execution errors return status 1.
Program output goes to stdout; diagnostics go to stderr.
`glime` selects the predictive stack; `glime-recursive` selects recursive descent.

## Lexical rules

- Identifiers are ASCII letters followed by ASCII letters or digits.
- Names and keywords are case-sensitive. Keywords are lowercase:
  `const var procedure call begin end if then while do odd write`.
- Numbers are unsigned decimal literals in `0..9223372036854775807`.
  A sign is an expression operator, not part of a literal.
- Whitespace is space, tab, CR, or LF. LF advances the line; columns count
  bytes starting at one, with a tab counting as one byte.
- `//` line comments and non-nested `/* ... */` comments are supported.
- Assignment is `:=`. Comparisons are `= # < <= > >=`; `#` means not equal.
  `!=` and `<>` are not alternatives.
- A program ends in a period, followed only by whitespace/comments and EOF.

## Grammar

This EBNF describes the accepted language, not either generator's input format.
Braces mean repetition; brackets mean optional syntax. `ident` and `number`
refer to the lexical rules above. A statement may be empty.

```text
program   = block "." .
block     = ["const" ident "=" number {"," ident "=" number} ";"]
            ["var" ident {"," ident} ";"]
            {"procedure" ident ";" block ";"}
            statement .
statement = [ ident ":=" expr
            | "call" ident
            | "begin" statement {";" statement} "end"
            | "if" condition "then" statement
            | "while" condition "do" statement
            | "write" expr ] .
condition = "odd" expr | expr ("=" | "#" | "<" | "<=" | ">" | ">=") expr .
expr      = ["+" | "-"] term {("+" | "-") term} .
term      = factor {("*" | "/") factor} .
factor    = ident | number | "(" expr ")" .
```

Multiplication/division bind more tightly than addition/subtraction. Binary
operators associate left: `19-6-2` is `(19-6)-2`, and `96/4/3` is `(96/4)/3`.
A leading sign applies to the first whole term: `-2*3+4` is `-(2*3)+4`.
Signs are allowed only at the start of an expression: use `2*(-3)`, not `2*-3`.
Conditions are not arithmetic values. There is no `else`, boolean connective,
parameter, return statement, input operation, array, or string type.

## Scope and execution

- Constants, variables, and procedures share a block-local namespace. Duplicate
  declarations in the same block fail; inner blocks may shadow outer names.
- All procedures in a block are mutually visible, including forward calls.
  Direct recursion and mutual recursion are supported. This is an explicit
  example dialect choice, not a claim about Wirth's original declaration rules.
- Procedure calls use the environment where the procedure was declared, not
  its caller's scope. Nested procedures capture the corresponding active
  enclosing invocation. Each call receives fresh variables initialized to zero.
- Only variables are assignable. Only procedures are callable. Only constants
  and variables may appear as identifier expressions.
- The entire AST, including uncalled procedures and unreachable branches, is
  name/category-checked before execution. Semantic failure produces no output.
- Values are signed 64-bit integers. Arithmetic uses Go's int64 wraparound;
  division truncates toward zero. Division by zero is a positioned runtime
  error. The minimum int64 divided by `-1` remains the minimum int64, as in Go.
- `odd` is true for positive or negative integers not divisible by two.
- `write expr` prints the decimal value followed by a newline. It is an output
  extension. The comment forms and machine-integer behavior are also explicit
  choices of this example, not historical-compatibility promises.
- Runtime and output-writer failures stop execution. Output already written is
  not rolled back. There are no execution quotas or recursion limits beyond
  Go's runtime and available resources: this is not a sandbox for untrusted code.

## Shared API and files

| File/package | Responsibility |
| --- | --- |
| [`lang/ast.go`](../../examples/pl0/lang/ast.go) | `Token`, `Block`, `Constant`, `Procedure`, `Statement`, `Expr`, `Condition`. Tokens retain source spelling/positions; ASTs contain no parser state or evaluator closures. |
| [`lang/lexer.go`](../../examples/pl0/lang/lexer.go) | `Lex(source string) ([]Token, error)`, ending in an explicit positioned EOF token. Symbolic terminal names are independent of numeric generated token IDs. |
| [`lang/interpreter.go`](../../examples/pl0/lang/interpreter.go) | `Execute(block *Block, output io.Writer) error`. Accepts a parser-produced AST, checks it, then executes it with fresh runtime storage; does not mutate the AST. |
| [`lalr/`](../../examples/pl0/lalr/) | `Parse(source string) (*lang.Block, error)`; shared lexer, glcc push parser, explicit EOF. |
| [`ll1/`](../../examples/pl0/ll1/) | Same `Parse` signature; shared lexer, glime complete-input parser, implicit EOF. Right-recursive arithmetic tails fold left into shared AST nodes. |
| [`recursive/`](../../examples/pl0/recursive/) | Same `Parse` signature and shared lexer; recursive descent generated from `ll1/grammar.y`, in a separate package with the same AST-building actions. |

Both grammars use Go actions solely to construct ASTs, not to execute the PL/0
program. Only the interpreter implements language execution. All adapters stop
on the first syntax error, returning no AST. Private typed hook panics carry
source locations across generated parser APIs; the adapters recover only that
type and re-panic unrelated failures. At glime's implicit EOF, the adapter uses
the lexer's final position.

The recursive frontend consumes Go call-stack space for nested grammar calls
and right-recursive lists. No fixed safe input depth is promised; stack or memory
exhaustion may terminate the process. See the [glime limits](glime.md).

Generated files import the example's shared language package, not generator
internals. Production example packages depend only on each other and the Go
standard library. Regeneration tests use generator APIs; no C compiler or
network is needed for the example's ordinary tests. See
[development checks](../development/README.md) for the separate C baseline gates.
