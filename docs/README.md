# Documentation

`lemon` turns grammars into **C parsers**; the equivalent `glcc` and `glemon`
commands generate **Go parsers** using Go actions. Choose a path by what you
need, rather than reading every page in order. This documentation follows
[Diátaxis](https://diataxis.fr/): learning, tasks, factual lookup, and understanding
have separate homes.

## Tutorials — learn by doing

- [Your first C parser](tutorials/first-parser.md): generate, compile, and run a
  small parser, with all example files supplied.
- [Your first Go parser](tutorials/go-parser.md): generate and run a calculator
  with typed Go actions and a Go caller.
- [Your first LL(1) Go parser](tutorials/ll1-parser.md): check, explain, and
  generate a predictive calculator with the separate `glime` command.
- [Build a PL/0 AST with glcc](tutorials/pl0-glcc.md): generate a LALR parser,
  inspect the tree, and execute it with the shared interpreter.
- [Build a PL/0 AST with glime](tutorials/pl0-glime.md): check a predictive
  grammar and build the same AST for that same interpreter.

## How-to guides — complete a task

- [Build and install Lemon](how-to/install.md)
- [Generate C from an existing grammar](how-to/generate-parser.md)
- [Use a custom parser template](how-to/custom-template.md)
- [Troubleshoot parser generation](how-to/troubleshoot.md)
- [Run PL/0 with either Go parser](how-to/run-pl0.md)

## Reference — look something up

- [Command-line interface](reference/cli.md)
- [Go generators and runtime API](reference/go-parser.md)
- [glime LL(1) command and runtime API](reference/glime.md)
- [PL/0 example language, shared AST, and interpreter](reference/pl0.md)
- [Grammar syntax](reference/grammar.md)
- [Declarations and preprocessor directives](reference/directives.md)
- [Output files and parser API](reference/output-files.md)
- [Compatibility boundary](reference/compatibility.md)

## Explanation — understand why and how

- [How Lemon works](explanation/how-lemon-works.md)
- [Why glime uses deferred completion](explanation/ll1.md)
- [Parser templates](explanation/parser-templates.md)
- [Lemon versus yacc and Bison](explanation/lemon-vs-yacc.md)

## Development and evidence

[Maintainer documentation](development/README.md) contains validation commands,
reference provenance, C-to-Go mappings, milestone records, and detailed
compatibility evidence. These are engineering records, not prerequisites for
using the generator. Their original paths remain stable for existing links.
