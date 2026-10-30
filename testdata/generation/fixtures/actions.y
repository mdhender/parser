%name ActionParser
%token_prefix TK_
%include {
/* Replace the template header, keeping line accounting. */
#include <stdlib.h>
}
%token_type {int}
%default_type { long }
%type number {int}
%type pair {double}
%type choice {int}
%token_destructor { dispose($$); }
%default_destructor { drop($$); }
%destructor number { same($$); }
%destructor choice { same($$); }
%destructor pair { distinct($$); }
%extra_argument { void *argument }
%extra_context { int context }
%stack_size 0
%realloc resize
%free release
%syntax_error { syntax(TOKEN); }
%parse_accept { accepted(); }
%parse_failure { failed(); }
%stack_overflow { overflow(); }
%code { /* extra code */ }
%fallback ID WORD .
%wildcard ANY .
start ::= seq.
seq ::= number pair choice empty.
number(A) ::= INTEGER(A). { use(A); }
pair(A) ::= LEFT(B) RIGHT(C). { A = B + C + @B; }
choice(A) ::= ID(B). { A = B; /*A-overwrites-B*/ }
choice(A) ::= WORD(B). { A = B; /*A-overwrites-B*/ }
empty(A) ::= . { A = 7; }
