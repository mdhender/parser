# Go generators and runtime API

`glcc` and `glemon` generate Go parsers from Lemon grammars containing **Go**
actions. They are equivalent thin wrappers around the shared `RunGo` entry
point. They reuse the grammar frontend, automaton, and table machinery, but
have a Go-specific emitter and runtime. `lemon` still generates C; it has no
language flag. For a complete example, see the
[Go calculator tutorial](../tutorials/go-parser.md).

## Commands and outputs

```text
glcc [options] grammar.y
glemon [options] grammar.y
```

Build from a checkout with `go build ./cmd/glcc` or `go build ./cmd/glemon`.
Use Go's flag syntax, placing **all options before the single grammar
filename**. String flags accept `-name value` or `-name=value`, not Lemon's
attached `-dDIR`, `-TFILE`, or `-DNAME` forms.

| Option | Default | Effect |
| --- | --- | --- |
| `-package NAME` | `parser` | Generated Go package name. |
| `-d DIR` | Input directory | Write outputs in an existing directory, using the grammar basename. |
| `-T FILE` | Embedded Go template | Use exactly this Go `text/template` skeleton. Explicit read/template errors fail; there is no fallback for an invalid override. |
| `-q` | `false` | Suppress the state report, not generated Go or diagnostics. |
| `-c` | `false` | Disable table compression; packed table construction still runs. |
| `-D NAME` | None | Define a grammar preprocessor name; repeat for multiple names. |

```sh
mkdir -p generated
glcc -package parser -d generated -D FEATURE -D DEBUG grammar.y
glemon -q -c -T templates/parser.go.tmpl grammar.y
```

Ordinary generation produces `grammar.go` and `grammar.out`; `-q` omits the
latter. Without `-d`, output names follow the input path. Token constants are
in the Go file, with `%token_prefix` applied; no `.h` or `.c` is produced.
These commands do not expose all of `lemon`'s options. Consult the
[C command-line reference](cli.md) separately for C flags. Check the command's
exit status: suppression does not delete reports left by an earlier run.

## Types and actions

| Declaration | Go meaning |
| --- | --- |
| `%name Parse` | Parser type name; default `Parse`. Also determines constructor name `NewParse`. |
| `%token_type {int}` | Semantic value type supplied with tokens; default `int`. |
| `%default_type {T}` | Default nonterminal value type; defaults to `%token_type`. |
| `%type expr {T}` | Override one nonterminal's type, allowing heterogeneous values. |
| `%token_prefix TK_` | Prefix generated token constants. |
| `%include { ... }` | Go imports and declarations immediately **after** the generated package clause. Must not contain another `package` clause. |
| `%code { ... }` | Trailing Go declarations. |

Types and action bodies must be valid Go in the generated package. For example:

```lemon
%include {
import "strconv"
}
%token_type {string}
%type number {int}
input ::= number(N). { _ = N }
number(N) ::= INTEGER(S). { N, _ = strconv.Atoi(S) }
```

Aliases are ordinary typed local variables, not textual macros. Every alias in
a rule must be distinct, including a separate left-hand-side alias. The LHS
value starts at its type's **zero value** and remains zero unless assigned.
An actionless rule does not implicitly copy its first RHS value. Assign a
forwarded value explicitly, for example `expr(A) ::= INTEGER(B). { A = B }`.

## Runtime API and extra parameters

With default `%name Parse` and no extra parameters, the public API is:

```go
func NewParse() *Parse
func (p *Parse) Parse(token int, value int) // int is the default token_type.
func (p *Parse) Close()
```

`%name Calc` instead produces type `Calc`, constructor `NewCalc`, and methods
`(*Calc).Parse` and `(*Calc).Close`. The value parameter uses `%token_type`.
Supply one token at a time. Feed **token 0 for EOF**, with the semantic type's
zero value (for example `0`, `nil`, or `Token{}`), to finish parsing. Token
constants cover ordinary terminals; zero is reserved for EOF.

Invalid token codes (negative or beyond the generated terminal range) panic
before changing parser state. After acceptance or `Close`, the parser can be
reused for a new input stream.

`Close` destroys pending stack values. Use it when finished, including after
abandoning a partial parse. A zero-value parser is usable when no extra context
is required, for example `var p Parse`; otherwise use the constructor to supply
the context.

The parameter declarations use **Go syntax**, each with one named, nonvariadic
parameter. Argument and context names must differ; `TOKEN` and names beginning
with `_lemon` are reserved:

```lemon
%extra_context {ctx *Context}
%extra_argument {out *Result}
```

Together these change the default API to:

```go
func NewParse(ctx *Context) *Parse
func (p *Parse) Parse(token int, value int, out *Result)
```

Context is supplied at construction; the extra argument is supplied on each
`Parse` call. Both are available by name as locals in actions, hooks, and
destructors. Mutations through a pointer persist, such as `out.Value = 3` or
`*out = Result{}`. Rebinding a local (`out = other`, `ctx = other`) does **not**
update the stored parameter. Supply the intended extra argument on each call,
and keep referenced data valid until pending values have been destroyed.

## Hooks and stack limits

Hooks contain Go statements and have access to extra argument/context locals:

| Declaration | Purpose |
| --- | --- |
| `%syntax_error { ... }` | Syntax-error notification; `TOKEN` is the offending token's semantic value, of `%token_type`. |
| `%parse_accept { ... }` | Successful acceptance. |
| `%parse_failure { ... }` | Unrecoverable parse failure. |
| `%stack_overflow { ... }` | Stack limit exceeded. |

Only `TOKEN` is documented as the special syntax-error value; do not depend on
internal runtime variable names. `%stack_size` unset or `0` selects dynamic
growth. A positive integer caps **total stack entries, including the base
entry**, not merely the number of shifted tokens. C expressions and negative
sizes are not supported. `%realloc` and `%free` are rejected as C-only; the Go
runtime manages its own storage.

## Destructors and ownership

`%token_destructor`, `%default_destructor`, and symbol-specific `%destructor`
contain Go statements. In these bodies, `$$` means the symbol's **typed
semantic value**. Replacement occurs only outside Go literals and comments:

```lemon
%token_type {*Token}
%token_destructor { if $$ != nil { $$.Release() } }
```

- An **aliased RHS** value is owned by the action; the parser does not
  automatically destroy it on reduction, even if the action does not use it.
  The action must transfer or release any resources it owns.
- An **unaliased RHS** value is automatically destroyed on reduction.
- Popping or discarding symbols, including during error recovery, and `Close`
  destroy the affected values.
- Successful acceptance drops the start symbol **without invoking its
  destructor**, matching Lemon. Arrange ownership of its result accordingly.

Go garbage collection does not replace destructor actions needed for resources
such as open files or application-managed handles.

## Shared grammar features and boundaries

Ordinary [grammar syntax](grammar.md) and frontend
[directives](directives.md), including precedence, conditional preprocessing,
fallback, wildcard tokens, and `error`-symbol recovery, are supported. Their
embedded code and types must use the Go rules above, not the C examples in
those shared pages. C actions, C types, and C runtime/preprocessor macros are
not translated or supported. Grammar `%if`/`%ifdef` processing with `-D` is
distinct from embedding C `#define` directives in output. As in the pinned C
runtime, disabling compression can change when reductions and error recovery
occur on malformed input; it is not an identical-recovery-timing promise.

The generated runtime uses only the Go standard library; dependencies introduced
by your own actions or imports are your responsibility. The
[pinned C differential evidence](../COMPATIBILITY.md) applies to the C generator,
not a byte-for-byte or ABI compatibility promise for generated Go.

## Go templates

The embedded default is `internal/lemon/lempar.go.tmpl`. `-T FILE` overrides the
**full Go runtime skeleton**, using Go's `text/template` language. It is not a
C `lempar.c` template and does not use the C template's `%%` section ABI. Do not
pass a C template to `glcc` or `glemon`.

Start from the embedded Go skeleton when making an override. Its template data
fields and helper contract are **internal and not stable** across releases;
overrides must track the generator version. The result must be valid Go, with
the generated package and semantic declarations in the right places. Most
grammars should use the embedded template and `%include` / `%code` rather than
maintaining a private runtime.
