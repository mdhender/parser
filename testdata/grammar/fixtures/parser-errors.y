%unknown value .
start bad .
good ::= TOKEN .
%left TOKEN .
%right TOKEN .
%wildcard ANY OTHER .
%fallback TOKEN FALL .
%fallback TOKEN FALL .
%token_class badclass lower .
other ::= TOKEN . [lower]
{ first(); }
{ second(); }
