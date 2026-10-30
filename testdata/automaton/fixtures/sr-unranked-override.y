// Non-null rule precedence symbol with prec=-1, and a ranked shift token.
%left OP .
start ::= expr .
expr ::= expr OP expr . [UNRANKED]
expr ::= ID .
