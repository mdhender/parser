// Non-null ranked rule precedence, but the shift token OP has prec=-1.
%left RANK .
start ::= expr .
expr ::= expr OP expr . [RANK]
expr ::= ID .
