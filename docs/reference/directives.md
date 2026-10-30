# Declaration and preprocessor reference

All declarations and conditional forms listed here are implemented in the Go
port of pinned SQLite 3.50.4 Lemon. The [compatibility evidence](../COMPATIBILITY.md)
records parsing, artifact, and runtime verification separately; declaration
support is not a claim that every C runtime configuration has been tested.

## Declaration argument conventions

Keywords are case-sensitive. Declarations begin with `%`; unknown keywords
diagnose. Single-argument declarations consume one identifier, quoted argument,
or brace block, with outer quotes/braces removed. **Do not add a period** unless
the syntax table explicitly shows one. Use braces for C text, types containing
spaces, and values such as negative stack sizes:

```text
%name Calculator
%token_type {int}
%stack_size {-1}
%token NUMBER PLUS .
```

Repeated text declarations generally concatenate rather than replace their
arguments; separators are not automatically supplied for non-code text.
Repeated `%include` and `%code` are useful; repeated names/types/context text
are not a way to override earlier values. `%type` rejects a duplicate explicit
type. Source line directives are inserted into appropriate C text unless `-l`
is set.

## Parser identity and inserted C

| Declaration | Meaning/default |
| --- | --- |
| `%name NAME` | Generated API prefix, default `Parse`. Template copying rewrites eligible `Parse` prefixes to this name. |
| `%token_prefix PREFIX` | Prefix generated token constants, default empty. `%token_prefix TK_` yields names such as `TK_NUMBER`. |
| `%include {C}` | Append C text near the beginning of generated `.c`. Does not read another file. A literal `#include` inside the text is handled later by the C compiler. |
| `%code {C}` | Append C text at the end of generated `.c`. |
| `%extra_argument {DECL}` | Extra argument on each parse call, stored/fetched through generated `ARG_*` macros. Example: `%extra_argument {struct Context *ctx}`. |
| `%extra_context {DECL}` | Context passed during construction/initialization, stored/fetched through generated `CTX_*` macros. Example: `%extra_context {struct Arena *arena}`. |

Context/argument declarations should end in the parameter's identifier; the
emitter extracts that trailing name. They are declarations, not arbitrary C
expressions or lists of parameters.

## Types and destruction

| Declaration | Meaning/default |
| --- | --- |
| `%token_type {TYPE}` | Shared semantic C type for all terminals; default `void*`. |
| `%default_type {TYPE}` | Default nonterminal semantic type. If absent, untyped nonterminals use semantic slot zero. |
| `%type symbol {TYPE}` | Explicit nonterminal semantic type. Duplicate explicit type diagnoses. The parser accepts a symbol name, but this is not a mechanism for separate terminal types. |
| `%token_destructor {C}` | Destructor for terminal values. |
| `%default_destructor {C}` | Destructor for nonterminals lacking a specific destructor. |
| `%destructor symbol {C}` | Specific nonterminal destructor. Terminal destruction uses `%token_destructor`, even if per-symbol destructor text was stored for a terminal. |

In destructor text, `$$` substitutes the relevant semantic union field:

```text
%type expr {struct Node *}
%destructor expr { node_free($$); }
```

Destructors are used when the parser discards values, including applicable
unused RHS values, recovery, and stack cleanup. Values referenced by actions
are the application's responsibility; these declarations do not provide
automatic ownership inference for arbitrary C code.

## Runtime hooks and stack configuration

| Declaration | Meaning/default |
| --- | --- |
| `%syntax_error {C}` | Syntax-error hook. The default template exposes the offending semantic value as `TOKEN`. |
| `%parse_accept {C}` | Acceptance hook. |
| `%parse_failure {C}` | Failure hook; invocation depends on the template's error-recovery compilation paths. |
| `%stack_overflow {C}` | Stack-overflow hook. |
| `%stack_size SIZE` | Stack-depth text; default `100`, guarded by `#ifndef YYSTACKDEPTH`. Nonpositive depth enables growth. Use braces for a negative value. |
| `%realloc FUNCTION` | Stack reallocator, default C `realloc`. |
| `%free FUNCTION` | Stack deallocator, default C `free`. |

Providing **both** `%realloc` and `%free` sets `YYDYNSTACK=1`, allowing growth
even with a positive initial stack depth. The custom functions must satisfy
the selected C template's allocation contract. These control stack storage,
not the allocator argument used to allocate the parser object itself.

Runtime C definitions placed in `%include`, such as `YYNOERRORRECOVERY`,
`YYSTACKDEPTH`, or `YYMALLOCARGTYPE`, are C/template configuration, not extra
Lemon declarations. A custom template can change the runtime interface and
hook behavior.

## Symbols, precedence, and token matching

| Declaration | Meaning/constraints |
| --- | --- |
| `%start_symbol nonterminal` | Select start LHS; default is original first rule's LHS. The start symbol must have rules and must not appear on any RHS. |
| `%token TOKEN ... .` | Predeclare terminals, controlling first-seen numeric order. Optional before ordinary use. |
| `%left TOKEN ... .` | Assign a new precedence level and left associativity. |
| `%right TOKEN ... .` | Assign a new precedence level and right associativity. |
| `%nonassoc TOKEN ... .` | Assign a new precedence level; an equal-precedence shift/reduce choice becomes an error action. |
| `%fallback BASE TOKEN ... .` | Each token after `BASE` retries as `BASE` when no usable action exists. First argument is the destination. A token cannot receive multiple fallback assignments. |
| `%wildcard TOKEN .` | Set one wildcard terminal. Additional wildcard names diagnose. |
| `%token_class name TOKEN ... .` | Define a new lowercase named multiterminal. Name must not already exist. Members must be uppercase terminals; adjacent `\|` or `/` prefixes are accepted. |

Later precedence declarations have higher precedence. All tokens on one
declaration share its level. Duplicate token precedence diagnoses.
See [grammar syntax](grammar.md#precedence-and-associativity) for conflict rules,
post-rule `[TOKEN]`, and inline multiterminals.

```text
%token ID SELECT INSERT ANY .
%fallback ID SELECT INSERT .
%wildcard ANY .
%token_class keyword SELECT INSERT .
```

Here `SELECT` and `INSERT` fall back to `ID`, not the other way around. `keyword`
matches either at one RHS position; it is not an additional terminal constant.

## Conditional preprocessing

Preprocessing runs over raw file text **before** grammar scanning, comments, or
C code blocks. Directives must start at column one. Excluded bytes and directive
lines become spaces, preserving newlines and source line numbers.

| Form | Recognition and effect |
| --- | --- |
| `%if EXPR` | Include following text when expression is true. A literal space after `if` is required. |
| `%ifdef EXPR` | Same evaluator as `%if`, not restricted to a single name. Literal space required. |
| `%ifndef EXPR` | Include when expression is false. Literal space required. |
| `%else` | Toggle inclusion at the current excluded depth. Must be followed by whitespace (a newline suffices). Trailing text is blanked, not evaluated. |
| `%endif` | End the excluded depth when applicable. Must be followed by whitespace. Trailing annotations are blanked. |

An EOF immediately after `%else` or `%endif`, with no following whitespace,
does not satisfy recognition. Indented forms are not preprocessor directives.
Conversely, column-one directive-looking text inside a C block or comment is
still processed in this initial pass.

### Expression vocabulary

| Form | Meaning |
| --- | --- |
| `FEATURE` | True if defined by `-DFEATURE`; false otherwise. Case-sensitive. |
| `!FEATURE` | Negation; repeated `!` toggles negation. |
| `A && B` | Boolean AND with Lemon's early-return behavior below. |
| `A \|\| B` | Boolean OR with Lemon's early-return behavior below. |
| `(EXPR)` | Recursively evaluated subexpression. |

Names begin with an ASCII letter and continue with letters, digits, or `_`.
Definitions come from the CLI; `-DFEATURE=0` still defines `FEATURE` as true.
There is no value expansion, numeric literal arithmetic, comparison, C
`defined(...)`, `%define`, `%undef`, or `%elif` syntax. `-U` changes the CLI
definition list before preprocessing; it is not a file directive.

```text
%if FEATURE && !OMIT_FEATURE
statement ::= FEATURE_TOKEN .
%else
statement ::= BASIC_TOKEN .
%endif
```

For this fragment, `lemon -DFEATURE grammar.y` selects `FEATURE_TOKEN`;
`lemon -DFEATURE -DOMIT_FEATURE grammar.y` selects `BASIC_TOKEN`.
`lemon -E grammar.y` prints the actual preprocessed text without parsing it.

### Pinned evaluator quirks

Do not assume C operator precedence. The evaluator can short-circuit the
**entire remaining expression**, not only one operand:

| Expression, with `A` defined and `MISSING` undefined | Result |
| --- | --- |
| `MISSING && A \|\| A` | False: returns at `&&` before reaching `\|\|`. |
| `(MISSING && A) \|\| A` | True: parentheses limit the early return to the subexpression. |
| `A \|\| @ invalid tail` | True: skipped suffix is not syntax-checked. |

Expressions inside already excluded nested conditionals need not be evaluated
or syntax-checked. Only names actually evaluated are marked used for the C
prologue's definition list. Conditional nesting tracks excluded regions rather
than a general active-branch stack: unmatched active `%endif` and an
unterminated included branch are not uniformly rejected; an excluded branch
left open does diagnose. Use balanced directives and explicit parentheses
rather than relying on these quirks.

## Related documentation

- [CLI definitions and undefinitions](cli.md)
- [Grammar syntax](grammar.md), [output files](output-files.md)
- [Custom templates](../how-to/custom-template.md)
- [Parser templates explained](../explanation/parser-templates.md)
