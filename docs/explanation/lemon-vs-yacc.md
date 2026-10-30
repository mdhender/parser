# Lemon versus yacc and Bison

**Which differences matter when bringing a yacc or Bison mental model to Lemon?**

Lemon, traditional yacc, and GNU Bison all let a grammar describe how tokens
form larger constructs, with actions that compute semantic values. Their
similarities make the basic shift/reduce model transferable. Their differences
make a mechanical grammar or API substitution unsafe.

Here, “traditional yacc” means the familiar pull-parser interface and yacc
grammar conventions. **GNU Bison is not limited to that interface or feature
set.** Some historical Lemon documentation contrasts Lemon with yacc and Bison
together; those broad comparisons should not be read as a description of all
current Bison configurations.

## Control flow is a default, not a capability contest

With Lemon's standard template, the application supplies a token to a parser
instance on each call. The parser retains its stack between calls and does not
ask a lexer for more input. This push interface naturally fits applications
that already own token production or need to interleave independent parses.

The traditional yacc model calls a parsing function that repeatedly calls a
lexer until the parse finishes. Bison normally generates such a **pull** parser,
but also supports **push**, or combined push/pull, interfaces. Its pure push
parser maintains an explicit parser-state object and supports multiple
instances. Push parsing is therefore a built-in Lemon convention, not a feature
Bison lacks. See Bison's [push-parser documentation](https://www.gnu.org/software/bison/manual/html_node/Push-Decl.html).

Reentrancy and push/pull control are related integration choices, not synonyms.
Bison can generate pure (reentrant) parsers. Lemon stores parsing state in
instances, but application actions and shared facilities still need their own
thread-safety discipline. In particular, “Lemon has no globals” is not a sound
claim about the entire toolchain: the original C generator has process-static
mutable state, and the standard runtime has shared debug-tracing state. The Go
port makes generator state per invocation; that does not rewrite the C runtime.

## LALR(1) is Lemon's algorithm, not all of Bison

Lemon builds deterministic LALR(1) tables. Traditional yacc uses LALR(1), and
Bison uses LALR(1) by default. However, Bison also supports IELR and canonical
LR table construction, as well as GLR parsing. These are meaningful choices,
not alternate spellings for the same machine. Bison's
[LR table construction discussion](https://www.gnu.org/software/bison/manual/html_node/LR-Table-Construction.html)
explains the distinctions.

LALR state sharing can introduce conflicts absent from a canonical LR(1)
machine. A grammar that works under a different Bison configuration is not
therefore guaranteed to be conflict-free in Lemon. Lemon does not branch into
multiple live parses to resolve ambiguity at runtime. Precedence, associativity,
and grammar structure determine its deterministic choices. See
[how Lemon works](how-lemon-works.md) for the underlying compilation model.

## Semantic names help, but are not unique to Lemon

Lemon's grammar actions use aliases attached to production symbols. A useful
way to think about them is as names for the semantic inputs and result of a
reduction, rather than numbered stack positions. Inserting another symbol need
not force a renumbering of every reference in the action.

Traditional yacc notation emphasizes positional references such as `$1` and
`$$`. Bison supports those and also
[named references](https://www.gnu.org/software/bison/manual/html_node/Named-References.html).
The distinction is Lemon's syntax and conventions, not an inability to name
values in Bison. Token naming, rule punctuation, declarations, and action
rewriting also differ; shared concepts do not make the grammar formats
interchangeable. The [grammar reference](../reference/grammar.md) defines
Lemon's notation without requiring a yacc translation model.

## Precedence can change the meaning of an apparently similar rule

Both systems can resolve a shift/reduce conflict by comparing the lookahead
token's precedence with the reducing rule's precedence. But default rule
precedence is not inferred the same way.

In the pinned Lemon implementation, an explicit rule-precedence mark takes
priority; otherwise, the rule takes the **first precedence-bearing symbol** on
its right-hand side, scanning left to right (and scanning multiterminal
constituents in order). In Bison, default rule precedence comes from the
**last terminal** in the rule, with an explicit override available. See
[Bison's precedence explanation](https://www.gnu.org/software/bison/manual/html_node/How-Precedence.html).

This difference matters especially for rules containing multiple operators.
Copying precedence declarations while leaving rule structure unchanged can
change conflict resolution. A comparison must examine the resulting decisions,
not just whether both generators accept the grammar. Exact Lemon declarations
belong in the [directives reference](../reference/directives.md).

## Destructors express ownership, with different conventions

Lemon supports destructors for semantic values discarded during recovery or
parser cleanup. Its action translator also generates cleanup for unreferenced
right-hand-side values during ordinary reductions. When an action uses a value,
that action is responsible for its ownership, often by transferring it into a
larger object.

Bison also supports [destructors for discarded symbols](https://www.gnu.org/software/bison/manual/html_node/Destructor-Decl.html),
including per-symbol, per-type, and default policies. It is incorrect to present
destructors as something Bison cannot do. The important migration question is
*when* a value is considered used or discarded and who must release it. The
presence of similarly named destructor directives does not establish identical
cleanup semantics during successful reductions or error recovery.

## This Go port changes the generator, not the comparison's runtime

The comparison above concerns Lemon's generated C parser using its standard
template. This repository implements the generator in Go while preserving the
pinned SQLite algorithms, action encodings, and C output conventions. It is
not an attempt to add Bison's alternative parsing algorithms or accept Bison
grammars.

One intentional packaging difference is the port's embedded pinned template,
used only as the final fallback after external template discovery. Original
Lemon needs an external template. Neither that fallback nor the Go
implementation language changes the parser's push interface or LALR(1) model.
See [parser templates](parser-templates.md) for that boundary.

## Sources and scope

The SQLite [Lemon overview](https://www.sqlite.org/lemon.html) and
[pinned manual](https://sqlite.org/src/doc/4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3/doc/lemon.html)
explain Lemon's design motivations. The Bison manual sections linked above
qualify historical comparisons with current Bison capabilities. Local behavior
was checked against `FindRulePrecedences` in the pinned
[`lemon.c`](../../reference/sqlite/tool/lemon.c), its Go
[translation](../../internal/lemon/grammar.go), the
[action translator](../../internal/lemon/code.go), and the pinned
[`lempar.c`](../../reference/sqlite/tool/lempar.c) runtime.

These conceptual similarities are not a cross-tool compatibility guarantee.
Consult the [compatibility reference](../reference/compatibility.md) and existing
[verification record](../COMPATIBILITY.md) for this port's tested scope. The
[architecture](../ARCHITECTURE.md) and
[development index](../development/README.md) retain maintainer-level detail.
