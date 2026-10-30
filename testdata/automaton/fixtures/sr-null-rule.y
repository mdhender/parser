// Concatenation has no terminal and therefore a NULL rule precedence symbol.
start ::= expr .
expr ::= expr expr .
expr ::= ID .
