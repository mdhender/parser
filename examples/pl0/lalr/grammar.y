%name Parser
%token_prefix TK_
%token_type {lang.Token}
%include {
import "github.com/mdhender/parser/examples/pl0/lang"
}
%extra_argument {result **lang.Block}
%syntax_error { panic(syntaxError{TOKEN}) }
%parse_failure { panic(syntaxError{}) }
%type program {*lang.Block}
%type block {*lang.Block}
%type constants {[]lang.Constant}
%type const_list {[]lang.Constant}
%type variables {[]lang.Token}
%type ident_list {[]lang.Token}
%type procedures {[]lang.Procedure}
%type statement {*lang.Statement}
%type statements {[]*lang.Statement}
%type condition {*lang.Condition}
%type relation {lang.Token}
%type expr {*lang.Expr}
%type term {*lang.Expr}
%type factor {*lang.Expr}
%type sign {lang.Token}

program(A) ::= block(B) DOT. { A = B; *result = A }
block(A) ::= constants(C) variables(V) procedures(P) statement(S). {
 A = &lang.Block{Constants:C, Variables:V, Procedures:P, Statement:S}
}
constants(A) ::= . { A = nil }
constants(A) ::= CONST const_list(B) SEMI. { A = B }
const_list(A) ::= IDENT(N) EQ NUMBER(V). { A = []lang.Constant{{Name:N, Value:V}} }
const_list(A) ::= const_list(B) COMMA IDENT(N) EQ NUMBER(V). { A = append(B,lang.Constant{Name:N,Value:V}) }
variables(A) ::= . { A = nil }
variables(A) ::= VAR ident_list(B) SEMI. { A = B }
ident_list(A) ::= IDENT(N). { A = []lang.Token{N} }
ident_list(A) ::= ident_list(B) COMMA IDENT(N). { A = append(B,N) }
procedures(A) ::= . { A = nil }
procedures(A) ::= procedures(B) PROCEDURE IDENT(N) SEMI block(C) SEMI. { A = append(B,lang.Procedure{Name:N,Block:C}) }
statement(A) ::= . { A = nil }
statement(A) ::= IDENT(N) ASSIGN expr(E). { A = &lang.Statement{Kind:"assign",Token:N,Expr:E} }
statement(A) ::= CALL IDENT(N). { A = &lang.Statement{Kind:"call",Token:N} }
statement(A) ::= BEGIN(B) statements(S) END. { A = &lang.Statement{Kind:"begin",Token:B,Body:S} }
statement(A) ::= IF(I) condition(C) THEN statement(S). { A = &lang.Statement{Kind:"if",Token:I,Condition:C,Body:[]*lang.Statement{S}} }
statement(A) ::= WHILE(W) condition(C) DO statement(S). { A = &lang.Statement{Kind:"while",Token:W,Condition:C,Body:[]*lang.Statement{S}} }
statement(A) ::= WRITE(W) expr(E). { A = &lang.Statement{Kind:"write",Token:W,Expr:E} }
statements(A) ::= statement(S). { A = []*lang.Statement{S} }
statements(A) ::= statements(B) SEMI statement(S). { A = append(B,S) }
condition(A) ::= ODD(O) expr(E). { A = &lang.Condition{Token:O,Left:E} }
condition(A) ::= expr(L) relation(O) expr(R). { A = &lang.Condition{Token:O,Left:L,Right:R} }
relation(A) ::= EQ(B). { A = B }
relation(A) ::= NE(B). { A = B }
relation(A) ::= LT(B). { A = B }
relation(A) ::= LE(B). { A = B }
relation(A) ::= GT(B). { A = B }
relation(A) ::= GE(B). { A = B }
expr(A) ::= term(B). { A = B }
expr(A) ::= sign(O) term(B). { A = &lang.Expr{Token:O,Right:B} }
expr(A) ::= expr(B) PLUS(O) term(C). { A = &lang.Expr{Token:O,Left:B,Right:C} }
expr(A) ::= expr(B) MINUS(O) term(C). { A = &lang.Expr{Token:O,Left:B,Right:C} }
sign(A) ::= PLUS(B). { A = B }
sign(A) ::= MINUS(B). { A = B }
term(A) ::= factor(B). { A = B }
term(A) ::= term(B) TIMES(O) factor(C). { A = &lang.Expr{Token:O,Left:B,Right:C} }
term(A) ::= term(B) SLASH(O) factor(C). { A = &lang.Expr{Token:O,Left:B,Right:C} }
factor(A) ::= IDENT(N). { A = &lang.Expr{Token:N} }
factor(A) ::= NUMBER(N). { A = &lang.Expr{Token:N} }
factor(A) ::= LPAREN expr(B) RPAREN. { A = B }
