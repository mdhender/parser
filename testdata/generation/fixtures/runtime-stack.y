%include {
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
struct Result { int value, accepted, errors, overflow, destroyed; };
}
%token_type {int}
%default_type {int}
%extra_argument {struct Result *result}
%ifdef FIXED
%stack_size 8
%else
%stack_size 0
%endif
%token_destructor { result->destroyed += $$; }
%syntax_error { result->errors++; }
%parse_accept { result->accepted++; }
%stack_overflow { result->overflow++; }
start ::= nested(A). { result->value = A; }
nested(A) ::= LEFT nested(B) RIGHT. { A = B + 1; }
nested(A) ::= NUMBER(B). { A = B; }
%code {
int main(void){
  struct Result r = {0};
  void *p = ParseAlloc(malloc);
#if YYSTACKDEPTH == 8
  /* Overflow pops seven LEFT values, then the parser can be reused. */
  int i;
  for(i=0; i<8; i++) Parse(p, LEFT, 1, &r);
  Parse(p, NUMBER, 37, &r);
  Parse(p, 0, 0, &r);
#else
  /* Right recursion forces growth well beyond the initial allocation. */
  int i;
  for(i=0; i<400; i++) Parse(p, LEFT, 0, &r);
  Parse(p, NUMBER, 37, &r);
  for(i=0; i<400; i++) Parse(p, RIGHT, 0, &r);
  Parse(p, 0, 0, &r);
#endif
  ParseFree(p, free);
  printf("value=%d accepted=%d errors=%d overflow=%d destroyed=%d\n",
    r.value, r.accepted, r.errors, r.overflow, r.destroyed);
  return 0;
}
}
