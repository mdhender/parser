%left LOW .
%left HIGH .
start ::= a .
start ::= b .
a ::= ID . [LOW]
b ::= ID . [HIGH]
