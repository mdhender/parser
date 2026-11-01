%name ParseTokens
%token_prefix TK_
%token_type {lang.Token}
%include {
import "github.com/mdhender/parser/examples/pl0/lang"
}
%syntax_error { fail(TOKEN) }
%type program {*lang.Block}
%type block {*lang.Block}
%type constants {[]lang.Constant}
%type const_list {[]lang.Constant}
%type const_tail {[]lang.Constant}
%type variables {[]lang.Token}
%type ident_list {[]lang.Token}
%type ident_tail {[]lang.Token}
%type procedures {[]lang.Procedure}
%type statement {*lang.Statement}
%type statements {[]*lang.Statement}
%type statement_tail {[]*lang.Statement}
%type condition {*lang.Condition}
%type relation {lang.Token}
%type expr {*lang.Expr}
%type term {*lang.Expr}
%type factor {*lang.Expr}
%type sign {lang.Token}
%type sum_tail {[]operation}
%type product_tail {[]operation}

program(A) ::= block(B) DOT. { A = B }
block(A) ::= constants(C) variables(V) procedures(P) statement(S). {
 A = &lang.Block{Constants:C, Variables:V, Procedures:P, Statement:S}
}
constants(A) ::= . { A = nil }
constants(A) ::= CONST const_list(B) SEMI. { A = B }
const_list(A) ::= IDENT(N) EQ NUMBER(V) const_tail(T). { A = append([]lang.Constant{{Name:N,Value:V}},T...) }
const_tail(A) ::= COMMA const_list(B). { A = B }
const_tail(A) ::= . { A = nil }
variables(A) ::= . { A = nil }
variables(A) ::= VAR ident_list(B) SEMI. { A = B }
ident_list(A) ::= IDENT(N) ident_tail(T). { A = append([]lang.Token{N},T...) }
ident_tail(A) ::= COMMA ident_list(B). { A = B }
ident_tail(A) ::= . { A = nil }
procedures(A) ::= PROCEDURE IDENT(N) SEMI block(B) SEMI procedures(T). { A = append([]lang.Procedure{{Name:N,Block:B}},T...) }
procedures(A) ::= . { A = nil }
statement(A) ::= . { A = nil }
statement(A) ::= IDENT(N) ASSIGN expr(E). { A = &lang.Statement{Kind:"assign",Token:N,Expr:E} }
statement(A) ::= CALL IDENT(N). { A = &lang.Statement{Kind:"call",Token:N} }
statement(A) ::= BEGIN(B) statements(S) END. { A = &lang.Statement{Kind:"begin",Token:B,Body:S} }
statement(A) ::= IF(I) condition(C) THEN statement(S). { A = &lang.Statement{Kind:"if",Token:I,Condition:C,Body:[]*lang.Statement{S}} }
statement(A) ::= WHILE(W) condition(C) DO statement(S). { A = &lang.Statement{Kind:"while",Token:W,Condition:C,Body:[]*lang.Statement{S}} }
statement(A) ::= WRITE(W) expr(E). { A = &lang.Statement{Kind:"write",Token:W,Expr:E} }
statements(A) ::= statement(S) statement_tail(T). { A = append([]*lang.Statement{S},T...) }
statement_tail(A) ::= SEMI statements(B). { A = B }
statement_tail(A) ::= . { A = nil }
condition(A) ::= ODD(O) expr(E). { A = &lang.Condition{Token:O,Left:E} }
condition(A) ::= expr(L) relation(O) expr(R). { A = &lang.Condition{Token:O,Left:L,Right:R} }
relation(A) ::= EQ(B). { A = B }
relation(A) ::= NE(B). { A = B }
relation(A) ::= LT(B). { A = B }
relation(A) ::= LE(B). { A = B }
relation(A) ::= GT(B). { A = B }
relation(A) ::= GE(B). { A = B }
expr(A) ::= sign(O) term(B) sum_tail(T). {
 A = B
 if O.Kind != "" { A = &lang.Expr{Token:O,Right:A} }
 A = fold(A,T)
}
sign(A) ::= PLUS(B). { A = B }
sign(A) ::= MINUS(B). { A = B }
sign(A) ::= . { A = lang.Token{} }
sum_tail(A) ::= PLUS(O) term(B) sum_tail(T). { A = append([]operation{{token:O,right:B}},T...) }
sum_tail(A) ::= MINUS(O) term(B) sum_tail(T). { A = append([]operation{{token:O,right:B}},T...) }
sum_tail(A) ::= . { A = nil }
term(A) ::= factor(B) product_tail(T). { A = fold(B,T) }
product_tail(A) ::= TIMES(O) factor(B) product_tail(T). { A = append([]operation{{token:O,right:B}},T...) }
product_tail(A) ::= SLASH(O) factor(B) product_tail(T). { A = append([]operation{{token:O,right:B}},T...) }
product_tail(A) ::= . { A = nil }
factor(A) ::= IDENT(N). { A = &lang.Expr{Token:N} }
factor(A) ::= NUMBER(N). { A = &lang.Expr{Token:N} }
factor(A) ::= LPAREN expr(B) RPAREN. { A = B }
