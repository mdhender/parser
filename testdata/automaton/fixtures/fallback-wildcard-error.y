%fallback ID KW .
%wildcard ANY .
start ::= list .
list ::= list item .
list ::= .
item ::= ID .
item ::= ANY .
item ::= error SEMI .
