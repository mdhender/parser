# Grammar syntax reference

Lemon consumes a text grammar and emits C. Names, rule order, token discovery
order, actions, and declarations affect its output. This page describes syntax;
[declarations and preprocessing](directives.md) have separate lookup tables.
The dialect is the project's pinned SQLite 3.50.4 Lemon dialect, not yacc syntax.

## Lexical forms

| Form | Meaning |
| --- | --- |
| `expr`, `input`, `expr_list` | Lowercase-initial nonterminal names. A rule LHS must begin with a lowercase letter. |
| `NUMBER`, `PLUS`, `ID` | Uppercase-initial terminal names. No declaration before use is required. |
| `value`, `A`, `left_value` | Action aliases may begin with either uppercase or lowercase letters. |
| `::=` | Rule separator; not `:` or `->`. |
| `.` | Rule terminator or list-declaration terminator. Not a C semicolon. |
| `// comment`, `/* comment */` | Comments outside code blocks. |
| `{ C text }` | Brace-delimited action/declaration text; nested braces, C comments and quoted literals are recognized by the scanner. |
| `"text"` | One declaration argument with outer quotes removed; top-level quoted arguments do not interpret escapes. |

Identifiers use ASCII letters and digits, with underscores allowed after the
initial letter. Classification is byte-oriented, not Unicode-aware. Do not use
an initial underscore as a symbol name. Tokens representing punctuation must
be named (for example `PLUS`), not written as yacc character literals such as
`'+'`. Whitespace separates tokens but is otherwise insignificant after
[conditional preprocessing](directives.md#conditional-preprocessing).

## Productions

```text
lhs ::= symbol symbol ... .
lhs(alias) ::= symbol(alias) ... . { C action }
lhs ::= .
```

A period finishes the RHS. The optional action follows the period. Alternatives
are separate productions, not yacc-style `lhs : A | B ;` branches.
An empty RHS is allowed; the RHS limit is 1,000 symbols per production.

```text
%token_type {int}
%left PLUS MINUS .
%left TIMES .

input ::= expr .
expr(A) ::= expr(B) PLUS expr(C). { A = B + C; }
expr(A) ::= expr(B) TIMES expr(C). { A = B * C; }
expr(A) ::= NUMBER(B). { A = B; }
```

This is a grammar fragment, not a tokenizer or complete application. The
original first rule's LHS is the default start symbol; `%start_symbol` can
override it. The selected start symbol must not occur on any RHS. Recursive
expression rules therefore normally sit below a separate start production.
Every nonterminal needs a defining production.

## Semantic aliases and actions

Aliases give C actions names for semantic values. `%token_type`, `%type`, and
`%default_type` determine the C types. An alias is not a grammar symbol or token
constant. Use distinct aliases in a rule; duplicate or unused aliases can
produce diagnostics during C emission, even when grammar parsing succeeded.

| Action syntax | Effect |
| --- | --- |
| `lhs(A) ::= TOKEN(B). { A = B; }` | Read the RHS semantic value and assign the LHS value. |
| `@B` in action text | RHS alias `B`'s major token code, rather than its semantic value. |
| `$$` in destructor text | The relevant semantic union field; not yacc-style general action notation. |
| `/*A-overwrites-B*/` | Exact action comment permitting direct LHS-alias `A` overwrite of the leftmost RHS-alias `B`. Later uses of `B` diagnose. |
| `{NEVER-REDUCE}` after a rule | Exact special marker setting the rule's never-reduce flag, not C action text. Does not itself remove the rule or add reduction code. |

There may be only one ordinary action block per rule. An explicit empty action
`{}` differs from no action: it is still present code and can affect rule
ordering and optimization. Alias rewriting is textual, not a full C lexer;
do not assume C comments or string literals isolate alias-like text from it.
Destructors dispose of values discarded by the parser; application actions
must manage values they consume. See [destructor declarations](directives.md#types-and-destruction).

## Precedence and associativity

```text
%left PLUS MINUS .
%left TIMES DIVIDE .
%right POWER .
%nonassoc EQUAL .

expr(A) ::= MINUS expr(B). [POWER] { A = -B; }
```

Each precedence declaration adds a level; later declarations are higher. Tokens
on the same declaration share a level and associativity. A token cannot be
assigned precedence twice. A rule inherits the **first** precedence-bearing
RHS symbol (including multiterminal constituents), not the last. A post-rule
`[TOKEN]` overrides that choice; only one precedence mark is allowed.

| Conflict | Resolution |
| --- | --- |
| Shift/reduce with missing precedence on either side | Shift; count an unresolved conflict. |
| Shift/reduce, unequal known precedence | Higher precedence wins. |
| Shift/reduce, equal precedence, `%left` | Reduce. |
| Shift/reduce, equal precedence, `%right` | Shift. |
| Shift/reduce, equal precedence, `%nonassoc` | Error action, rather than shift/reduce. |
| Reduce/reduce, unequal known rule precedence | Higher precedence wins. |
| Reduce/reduce, missing/equal precedence | Earlier rule wins; count an unresolved conflict. |

Use `-p` to include precedence-discarded actions in the `.out` report; it does
not change these resolutions.

## Multiterminals

A multiterminal matches one of several terminals at **one RHS position**. It
does not denote alternative productions or an arbitrary nonterminal choice.

```text
%token_class additive PLUS MINUS .
input ::= expr .
expr(A) ::= expr(B) additive(OP) expr(C). { A = combine(B, @OP, C); }
expr(A) ::= ID|NUMBER(B). { A = B; }
```

| Form | Constraint |
| --- | --- |
| `ID\|NUMBER` or `ID/NUMBER` | Inline compound; separator must be adjacent to the following identifier. `ID \|NUMBER` also scans as a compound; `ID \| NUMBER` does not. |
| `%token_class additive PLUS MINUS .` | New lowercase class name; must not already be a symbol. Members must be terminals. |
| `%token_class additive PLUS \|MINUS .` | Optional adjacent `\|` or `/` prefixes on class members are also accepted. |

Member order is observable; do not reorder classes or inline compounds when
comparing output. A class does not have its own terminal code in the generated
header.

## Fallback, wildcard, and recovery symbols

| Form | Meaning |
| --- | --- |
| `%fallback ID KEYWORD .` | If `KEYWORD` lacks a usable action, retry as `ID`. The first name is the destination. |
| `%wildcard ANY .` | Set the single wildcard terminal, used as a fallback match for otherwise unmatched non-EOF input where the template permits. |
| `error` on a RHS | Reserved lowercase error-recovery nonterminal. `ERROR` is an ordinary terminal. |

These facilities are implemented, including runtime fallback/wildcard and
recovery in the pinned template. Recovery behavior also depends on C macros
such as `YYNOERRORRECOVERY` and on compression; it is not specified by the
symbol name alone. A recovery production can be written as:

```text
statement ::= error SEMI .
```

## Input and diagnostic boundaries

The command reads one grammar file, limited to 100,000,000 bytes. A NUL byte
ends the effective input, as in pinned C. There is no general grammar-file
include directive: `%include` inserts C text into output.

The parser has no comprehensive final-state validation at EOF. Some incomplete
final constructs can be accepted rather than diagnosed. This is a pinned
quirk, not a recommended grammar form. Go bounds unfinished code-block scanning
at EOF rather than reproducing C out-of-bounds reads. Exact malformed-input
crash behavior is excluded from the compatibility claim.

## Related documentation

- [Declaration reference](directives.md), [CLI](cli.md), [output files](output-files.md)
- [Compatibility boundary](compatibility.md) and [verification evidence](../COMPATIBILITY.md)
- [First parser tutorial](../tutorials/first-parser.md)
- [How Lemon works](../explanation/how-lemon-works.md)
- [Lemon versus yacc](../explanation/lemon-vs-yacc.md)
