%name Calc
%token_prefix TK_
%token_type {int}
%type sum_tail {func(int) int}
%type product_tail {func(int) int}
input(A) ::= sum(B). { A = B }
sum(A) ::= product(B) sum_tail(C). { A = C(B) }
sum_tail(A) ::= PLUS product(B) sum_tail(C). { A = func(v int) int { return C(v+B) } }
sum_tail(A) ::= MINUS product(B) sum_tail(C). { A = func(v int) int { return C(v-B) } }
sum_tail(A) ::= . { A = func(v int) int { return v } }
product(A) ::= atom(B) product_tail(C). { A = C(B) }
product_tail(A) ::= TIMES atom(B) product_tail(C). { A = func(v int) int { return C(v*B) } }
product_tail(A) ::= . { A = func(v int) int { return v } }
atom(A) ::= NUM(B). { A = B }
atom(A) ::= LP sum(B) RP. { A = B }
