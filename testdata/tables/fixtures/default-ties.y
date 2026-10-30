// After TOKEN, left and right each reduce on two disjoint lookaheads.
// First encountered action wins the default-frequency tie, not rule index.
%token A B C D .
start ::= left A .
start ::= left C .
start ::= right B .
start ::= right D .
right ::= TOKEN .
left ::= TOKEN .
