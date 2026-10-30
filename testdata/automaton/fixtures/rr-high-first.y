%left LOW .
%left HIGH .
start ::= a .
start ::= b .
a ::= ID . [HIGH]
b ::= ID . [LOW]
