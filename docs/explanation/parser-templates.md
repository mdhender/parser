# Parser templates

**Why does Lemon need a template as well as a grammar?**

A grammar describes a language and its semantic actions. It does not, by itself,
provide the C machinery that stores a parsing stack, accepts tokens, handles
errors, and dispatches reductions. Lemon separates those concerns: the generator
computes the grammar-specific machine, while a **parser template** supplies the
runtime that executes it.

The standard template is `lempar.c`. It is input to the generator, not a library
the application loads at runtime. The generated `.c` combines template code,
tables, and grammar-supplied C code into one translation unit. A C compiler then
compiles that output as part of the application. The Go generator itself does
not need a C compiler to run.

## Three sources contribute to the generated parser

| Source | Contribution |
| --- | --- |
| Grammar | Productions, precedence, semantic types and actions, destructors, hooks, and application-specific C declarations. |
| Generator | Symbol and rule numbers, parsing states, packed tables, control definitions, and rewritten action code. |
| Template | Parser-instance storage, stack operations, table lookup, token entry point, reduction dispatch, recovery, and lifecycle functions. |

This division explains why changing only a template cannot turn Lemon into a
different grammar-analysis algorithm. The generator has already chosen LALR(1)
states and encoded their actions. Conversely, changing runtime behavior can
change how an application experiences the generated parser even when the tables
are unchanged. The template is part of the parser's semantics, not decoration.

The Go port preserves this division. Its emitter generates C for the pinned
SQLite template; the fact that the emitter is written in Go does not make an
arbitrary Go-language template compatible with it.

## The template is an ordered protocol

Lemon does not use a general-purpose named-placeholder engine. In the pinned
template, lines beginning with `%%` delimit successive sections. The generator
copies a section, inserts its next block of generated content, and continues.
The sequence includes grammar includes, token definitions, control macros and
semantic-value storage, tables, destructors, reduction actions, and error and
acceptance hooks.

The insertion points are **positional**, not named by the surrounding comments.
Comments make the file understandable to humans, but do not tell the emitter
which content belongs where. Moving, adding, or removing a separator can make
the emitter put valid C text in the wrong context. A custom template must also
agree with the emitted table encodings, macros, and stack/action conventions.

Template transfer performs a limited textual substitution for eligible `Parse`
prefixes when a grammar supplies a different parser name. It is not C-aware
identifier rewriting or a general macro processor. Separately, the action-code
translator rewrites grammar aliases into semantic-value accesses and supplies
destructor calls where appropriate. Neither mechanism parses C with a compiler
frontend, so arbitrary C text should not be assumed immune to rewriting.

## Selection is part of the input

For the pinned original Lemon and this port, an explicit template selection
takes precedence over automatic discovery. Without that selection, discovery
looks for a grammar-specific `.lt` file beside the grammar, then `lempar.c` in
the working directory, then the executable-directory/PATH lookup inherited
from Lemon. Executable-directory and PATH lookup are alternatives determined
by how the executable was invoked, not two unconditional search steps.

The **Go-port difference** comes at the end: if the final external candidate is
absent, the port uses an embedded copy of the pristine pinned `lempar.c`.
Original Lemon has no embedded fallback and cannot generate the parser without
an external template. The port embeds the actual pinned source file, not a
separately maintained approximation.

Embedding is a fallback, not an override. An external template found by the
existing lookup can still determine the result. An explicitly requested missing
template remains an error, and a selected-file read failure is not silently
replaced with the embedded template. This distinction lets the Go executable
work standalone without concealing a broken explicit template choice.

Consequently, a grammar alone does not completely identify a generated parser.
Template content, options, paths, and working directory also affect output.
Reproducible comparisons must control those inputs rather than assume that two
executables necessarily found the same `lempar.c`.

## Customization belongs at the appropriate layer

Many application choices do not require replacing the runtime. Semantic types,
actions, destructors, parser naming, and error hooks are grammar-level inputs.
The standard template already supports those variation points. The
[directives reference](../reference/directives.md) defines their exact forms.

A custom template is appropriate when the runtime mechanism itself must
change, such as its integration or stack-management behavior beyond existing
hooks. That is a tighter coupling to the generator than changing a grammar.
Output equality and runtime behavior verified with the pinned template do not
automatically extend to a replacement template.

For the practical customization workflow, see
[using a custom template](../how-to/custom-template.md). For option spelling and
artifact details, see the [CLI reference](../reference/cli.md) and
[output-files reference](../reference/output-files.md). Those pages describe
operation; this page explains why the template is a separate input.

## Sources and further reading

- SQLite's [pinned Lemon manual](https://sqlite.org/src/doc/4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3/doc/lemon.html),
  “Theory of Operation.”
- Pinned [`lempar.c`](../../reference/sqlite/tool/lempar.c) and
  [`lemon.c`](../../reference/sqlite/tool/lemon.c): `tplt_open`, `tplt_xfer`,
  and `ReportTable` define the selection and insertion protocol.
- Go [template selection and transfer](../../internal/lemon/template.go),
  [C emission](../../internal/lemon/emit.go), and the
  [embedded pinned source](../../reference/template.go).
- [How Lemon works](how-lemon-works.md) for the generator/runtime boundary;
  [architecture](../ARCHITECTURE.md), [reference provenance](../REFERENCE.md),
  and [development index](../development/README.md) for maintainer context.
