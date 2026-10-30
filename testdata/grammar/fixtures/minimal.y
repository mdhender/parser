/* An empty production and an actionless rule preceding an action rule. */
start ::= list .
list ::= list ITEM . {}
list ::= .
