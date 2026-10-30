%token_type {int}
%type left {long}
start ::= left right bad overwrite mismatch.
left(A) ::= X(B) Y(C). { A = B; }
right(A) ::= X(A) Y(A). { A = 1; }
bad(A) ::= Z(B). { int unused; }
overwrite(A) ::= W(B). { A = B; /*A-overwrites-B*/ B = 3; }
mismatch(A) ::= left(A). { A = 0; }
