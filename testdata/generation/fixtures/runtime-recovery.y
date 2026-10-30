%include {
#include <stdio.h>
#include <stdlib.h>
struct Result { int value, wild, recovered, accepted, errors, failed, tokens, values; };
}
%token_type {int}
%default_type {int}
%extra_argument {struct Result *result}
%token ID WORD ANY OTHER WILD SEMI BAD .
%fallback ID WORD .
%wildcard ANY .
%token_destructor { result->tokens += $$; }
%destructor value { result->values += $$; }
%syntax_error { result->errors++; }
%parse_accept { result->accepted++; }
%parse_failure { result->failed++; }
start ::= statements.
statements ::= statements statement.
statements ::= statement.
statement ::= value(A) SEMI. { result->value += A; }
statement ::= WILD ANY(A) SEMI. { result->wild += A; }
statement ::= error SEMI. { result->recovered++; }
value(A) ::= ID(B). { A = B; }
%code {
static void show(const char *name, struct Result *r){
  printf("%s value=%d wild=%d recovered=%d accepted=%d errors=%d failed=%d tokens=%d values=%d\n",
    name, r->value, r->wild, r->recovered, r->accepted, r->errors, r->failed, r->tokens, r->values);
}
int main(void){
  struct Result r = {0};
  void *p = ParseAlloc(malloc);
  /* WORD must fall back to ID; OTHER must match ANY only after WILD. */
  Parse(p, WORD, 7, &r);
  Parse(p, SEMI, 0, &r);
  Parse(p, WILD, 0, &r);
  Parse(p, OTHER, 11, &r);
  Parse(p, SEMI, 0, &r);
  /* BAD cannot match the wildcard here. Recover at SEMI and continue. */
  Parse(p, BAD, 13, &r);
  Parse(p, SEMI, 0, &r);
  Parse(p, ID, 17, &r);
  Parse(p, SEMI, 0, &r);
  Parse(p, 0, 0, &r);
  ParseFree(p, free);
  show("recovery", &r);

  r = (struct Result){0};
  p = ParseAlloc(malloc);
  Parse(p, ID, 19, &r);
  Parse(p, 0, 0, &r);
  ParseFree(p, free);
  show("failure", &r);

  r = (struct Result){0};
  p = ParseAlloc(malloc);
  Parse(p, ID, 31, &r);
  Parse(p, SEMI, 0, &r);
  Parse(p, ID, 23, &r);
  /* Compression reduces value before BAD; -c instead destroys the ID. */
  Parse(p, BAD, 29, &r);
  ParseFree(p, free);
  show("teardown", &r);
  return 0;
}
}
