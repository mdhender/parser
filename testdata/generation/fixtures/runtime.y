%include {
#include <stdio.h>
#include <stdlib.h>
struct Result { int value, accepted, errors; };
}
%token_type {int}
%default_type {int}
%extra_argument {struct Result *result}
%left MINUS .
%left TIMES .
%syntax_error { result->errors++; }
%parse_accept { result->accepted++; }
start ::= expr(A). { result->value = A; }
expr(A) ::= NUMBER(B). { A = B; }
expr(A) ::= expr(B) MINUS expr(C). { A = B - C; }
expr(A) ::= expr(B) TIMES expr(C). { A = B * C; }
%code {
int main(void){
  struct Result result = {0,0,0};
  void *p = ParseAlloc(malloc);
  Parse(p, NUMBER, 7, &result);
  Parse(p, MINUS, 0, &result);
  Parse(p, NUMBER, 2, &result);
  Parse(p, TIMES, 0, &result);
  Parse(p, NUMBER, 3, &result);
  Parse(p, 0, 0, &result);
  ParseFree(p, free);
  printf("value=%d accepted=%d errors=%d\n",result.value,result.accepted,result.errors);
  return result.value!=1 || result.accepted!=1 || result.errors!=0;
}
}
