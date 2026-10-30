# How Lemon works

**How does a grammar become a parser that an application can feed tokens?**

Lemon is a parser generator, not a tokenizer or an interpreter. It compiles a
grammar into C code that recognizes sequences of tokens and runs the grammar's
semantic actions. This repository ports the **generator** from C to Go. It still
generates **C parsers**, using the pinned SQLite runtime template; it does not
generate Go parsers or require Go in the application that uses its output.

There are two separate times to keep in mind. At generation time, Lemon reads
grammar text and builds a parsing machine. At application runtime, the generated
C code uses that machine to process tokens. The grammar-file scanner inside
Lemon is not the tokenizer for the language being recognized.

## Push parsing leaves the application in control

With the standard template, an application owns a parser instance and supplies
one token per call to its parsing entry point, called `Parse` by default. A
token consists of a numeric kind and a semantic value: for example, an integer
token's kind says “integer,” while its value carries the actual number or source
text. The tokenizer and the code that drives it belong to the application.

This is a **push interface**. The generated parser does not call a lexer to
obtain more input. Between calls, its instance retains the parsing stack, so the
application can decide when another token is available. A call can perform
several reductions before shifting the supplied token, reporting an error, or
accepting. Supplying token kind zero signals end of input; merely stopping calls
does not finish the parse.

The stack and semantic values belong to each parser instance. This makes
independent and nested parses possible without a single shared parsing stack.
It does not make arbitrary semantic actions, shared application data, or debug
tracing automatically thread-safe. Push parsing also means token-at-a-time
input, not automatic support for reparsing an edited document.

## What LALR(1) means here

Lemon builds a deterministic **LALR(1)**, or *look-ahead LR*, parser. It reads
tokens left to right and recognizes productions bottom-up. Conceptually, a
**shift** extends the recognized prefix by putting a token on the stack. A
**reduction** replaces the right-hand side of a production with its left-hand
side nonterminal and executes the production's semantic action. The “1” means
that the parsing decision uses one lookahead token, together with the state
already summarized by the stack; it does not mean the parser remembers only
one token of history.

A generator represents a partially recognized production as an *item*: a rule
with a dot marking how much has been recognized. A state groups items that can
be relevant at the same point. Closure adds productions for a nonterminal
after the dot; transitions move the dot past a symbol.

The pinned implementation constructs states from their LR(0) bases, identified
by rule and dot position without lookahead sets. It then propagates lookahead
sets between items until no set changes. Those are item-specific LALR
lookaheads, not simply one FOLLOW set per nonterminal. Sharing states with the
same core keeps tables compact, but can introduce conflicts that a full
canonical LR(1) machine would avoid. Lemon is not a generalized parser that
explores all competing parses.

## Compilation turns grammar decisions into numbers

The ordinary generation pipeline has several conceptual layers:

- **Grammar processing** preprocesses conditional sections, scans declarations
  and productions, and builds symbols, rules, types, and embedded action text.
  Embedded C remains text; Lemon is not a C compiler.
- **Grammar analysis** infers rule precedence and computes nullable symbols and
  FIRST sets, which describe possible beginnings of derived sequences.
- **Machine construction** builds states and transitions, propagates
  lookaheads, and assigns shift, reduce, accept, and error actions.
- **Conflict resolution** applies precedence and associativity where possible.
  An unresolved shift/reduce conflict retains the shift but is still diagnosed
  and makes generation fail. A produced file is not evidence of conflict-free
  generation.
- **Representation and emission** choose default reductions, combine eligible
  shift/reduce actions, reorder states, pack table entries, and insert tables
  and action code into a C template.

These are compilation stages, not operations repeated for every input token.
The resulting `.c` contains the runtime and grammar-specific data; the `.h`
provides token constants. The `.out` report exposes the machine for inspection,
not for loading at runtime. Exact artifact behavior belongs in the
[output-files reference](../reference/output-files.md).

## The C runtime interprets the compiled machine

The packed representation uses action and lookahead arrays, state offsets, and
default actions. An offset plus a symbol code selects a candidate slot; its
lookahead tag confirms whether that slot really belongs to the lookup. Packing
shares storage rather than allocating a full state-by-symbol matrix. It is not
a different parsing algorithm.

At runtime, a terminal lookup selects a shift, reduction, acceptance, or error.
After a reduction, a nonterminal lookup determines the next transition. The
template also implements fallback-token and wildcard lookup when the grammar
enables them. A reduction runs inserted C action code and combines semantic
values; constructing a syntax tree is a choice of those actions, not an
automatic Lemon result.

The same runtime manages stack storage, error handling, and calls to generated
destructors. Recovery depends on the grammar's `error` symbol and compile-time
settings; “syntax error” does not always mean immediate termination. Ownership
of values therefore matters even when the application expects valid input.

## The port preserves the boundary

The Go port translates grammar analysis, automaton construction, packing, and
emission rather than replacing them with another parsing strategy. Its mutable
generator state is per invocation, unlike the original generator's process
statics. That implementation distinction is separate from the generated C
parser's per-instance state. The port also embeds the pinned template as a
**final fallback** after external template discovery; original Lemon requires
an external template. See [parser templates](parser-templates.md).

Matching the algorithm alone is not the project's compatibility criterion:
ordering, diagnostics, paths, and emitted bytes matter too. The
[compatibility reference](../reference/compatibility.md) describes the public
boundary; the existing [compatibility record](../COMPATIBILITY.md) contains
verification evidence rather than a promise about every grammar or platform.

## Sources and further reading

- SQLite's [Lemon overview](https://www.sqlite.org/lemon.html) and
  [manual at the pinned revision](https://sqlite.org/src/doc/4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3/doc/lemon.html).
- The pinned [generator](../../reference/sqlite/tool/lemon.c), especially
  `FindFirstSets`, `FindStates`, `FindFollowSets`, and `ReportTable`, and
  [runtime template](../../reference/sqlite/tool/lempar.c), especially `Parse`
  and `yy_reduce`.
- The Go [generation flow](../../internal/lemon/lemon.go),
  [automaton](../../internal/lemon/automaton.go), and
  [table packing](../../internal/lemon/table.go).
- [Grammar reference](../reference/grammar.md) for notation;
  [architecture](../ARCHITECTURE.md) for the maintainer-level translation map;
  [development index](../development/README.md) for engineering documentation.
