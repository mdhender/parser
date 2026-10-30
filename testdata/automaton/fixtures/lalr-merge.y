// Canonical LR(1), not LALR(1): a/b completed configurations merge C contexts.
start ::= A a D .
start ::= B a E .
start ::= A b E .
start ::= B b D .
a ::= C .
b ::= C .
