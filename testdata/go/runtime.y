// Actions deliberately use syntax shared by C and Go. Parameter declarations
// and drivers are target-specific in scripts/test-go-runtime.sh.
%name Runtime
%token_prefix TK_
%token_type {int}
%left PLUS MINUS.
%left TIMES.
%fallback ID KEY.
%wildcard ANY.
%token OTHER BAD.
%syntax_error { *stats += 1 + TOKEN*10000; }
%parse_accept { *stats += 1000000; }
%parse_failure { *stats += 100000000; }
%token_destructor { *stats += $$*100; }
start ::= items.
items ::= .
items ::= items expr(A) SEMI. { *out += A; }
items ::= items error SEMI.
expr(A) ::= expr(B) PLUS expr(C). { A = B+C; }
expr(A) ::= expr(B) MINUS expr(C). { A = B-C; }
expr(A) ::= expr(B) TIMES expr(C). { A = B*C; }
expr(A) ::= LP expr(B) RP. { A = B; }
expr(A) ::= NUM(B). { A = B; }
expr(A) ::= ID(B). { A = B+10; }
expr(A) ::= ANY(B). { A = B+20; }
