# glime: LL(1) Go generator

`glime` is a new LL(1) companion to Lemon, not a recovered historical Lime
implementation. It is independent of `lemon` (C LALR) and `glcc`/`glemon` (Go
LALR). C reference compatibility does not establish glime compatibility.

## Command line

```sh
go build -o glime ./cmd/glime
./glime check grammar.y
./glime explain grammar.y
./glime -strategy=stack -package parser -d build generate grammar.y
```

All flags must precede the required subcommand. Exactly one grammar follows it.
Go flag forms `-flag value` and `-flag=value` work. Flags after the subcommand
are rejected. `-h` prints usage and succeeds; errors return status 1.

| Flag | Default | Meaning |
| --- | --- | --- |
| `-strategy` | `stack` | Predictive stack; `recursive` explicitly fails as not implemented |
| `-package` | `parser` | Generated Go package identifier |
| `-d` | grammar directory | Output directory, which must already exist |
| `-D` | none | Repeatable Lemon preprocessor definition; `NAME=value` defines `NAME` |

`check` analyzes eligibility and validates generated Go syntax without writing
files. It does not type-check user code or resolve its imports; build the output
with Go to validate those. `explain` also prints deterministic nullable, FIRST,
FOLLOW, SELECT and prediction information, including conflicts on invalid input.
Diagnostics go to stderr; explanation goes to stdout. Warnings do not fail the
command. `generate` writes only `<grammar-basename>.go`, formatted with gofmt,
replacing an existing file only after analysis and syntax validation succeed.
Token numbers follow source symbol order and can have gaps. Use named constants.

## Grammar boundary

Lemon scanning, preprocessing (`%if`, `%ifdef`, `%ifndef`, `%else`, `%endif`),
rule syntax, aliases, and declarations come from the existing frontend. Actions
contain Go, not C. First rule's LHS is the default start symbol; `%start_symbol`
overrides it. Empty RHS means epsilon.

Supported declarations: `%name`, `%token_prefix`, `%token`, `%start_symbol`,
`%token_type`, `%default_type`, `%type`, `%include`, `%code`, `%parse_accept`,
`%syntax_error`, `%parse_failure`.

Rejected rather than ignored:

- Direct/indirect left recursion, including through nullable prefixes;
  undefined and nonproductive nonterminals; overlapping production predictions.
- Precedence declarations and rule precedence annotations: they cannot resolve
  LL(1) conflicts. Express precedence in the grammar structure instead.
- Fallback, wildcard, `error` recovery, token classes and inline multiterminals
  (`A|B` / `A/B`), and `NEVER-REDUCE` annotations.
- Destructors, extra arguments/context, stack-size/overflow directives, and C
  allocator directives. Values use Go garbage collection. Pass application data
  as semantic values or action closures; no implicit parser context is supplied.

Unreachable productions receive warnings. glime does not eliminate left
recursion, left-factor, or otherwise rewrite the grammar. A conflict identifies
competing productions and overlapping lookahead tokens. FIRST/FIRST conflicts
usually need factoring; FIRST/FOLLOW conflicts require removing the overlap
between a nullable choice and the tokens that follow it.

## Generated API and semantic values

With `%name Calc` (default `Parse`):

```go
type CalcToken struct { Kind int; Value int } // Value uses %token_type
func Calc(tokens []CalcToken) (StartType, error)
```

This is a complete-input API, not the LALR generator's incremental API. Do not
append EOF: it is implicit (`$`, internal number 0). Negative, nonterminal,
unknown and explicit EOF token kinds are rejected. A successful call returns
the start value; a failed call returns its zero value and an error naming the
actual and expected tokens. There is no lexer or recovery. Calls have independent
stacks and frames; any shared state in user actions remains the user's concern.

Terminal values use `%token_type` (default `int`). Nonterminals use `%type`,
then `%default_type`, then `%token_type`, then `int`. Heterogeneous values live
in a generated typed struct, without reflection or generator dependencies.

LHS aliases start at their typed zero value. RHS aliases are typed local copies
of completed children. Duplicate aliases within a rule, Go keywords, `_`, and
the internal `_glime` prefix are rejected. Aliases need no substitution: Go
strings, raw strings and comments remain unchanged. No LHS alias means a
zero-valued result, even for an actionless single-child rule. There is no implicit
Lemon C union copy. Mutating a referenced slice/map/pointer has ordinary Go
semantics.

Actions run only after all RHS symbols complete, including epsilon actions.
Already-completed actions are not rolled back on later malformed input.
`%include` supplies imports and declarations after the package declaration;
`%code` supplies trailing declarations. The runtime reserves `_glime` names and
names prefixed with the parser name followed by `yy`.

`%parse_accept` runs once after the entire input is consumed. On the first error,
`%syntax_error` runs followed by `%parse_failure`, once each. Both error hooks
can read `TOKEN`, the unexpected semantic token value (zero at implicit EOF).
There is no alias scope or extra context in hooks, and no destructor calls.

## Verification

Offline tests compile and execute standalone generated parsers for arithmetic,
heterogeneous values, nested/right-recursive/epsilon productions, action order,
deep nesting, malformed token streams and hooks. Tests also exercise the CLI,
analysis diagnostics and deterministic generation. SQLite's C `parse.y` is not
an acceptance fixture for this Go backend. See the
[tutorial](../tutorials/ll1-parser.md) and [design](../explanation/ll1.md).
