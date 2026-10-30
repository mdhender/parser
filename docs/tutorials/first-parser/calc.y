%include {
#include <stdio.h>
#include <stdlib.h>
}
%token_type {int}
%syntax_error { fprintf(stderr, "Syntax error\n"); exit(1); }
%parse_failure { fprintf(stderr, "Parse failed\n"); exit(1); }

input ::= sum.
sum ::= INTEGER(A) PLUS INTEGER(B). { printf("Result: %d\n", A + B); }
