// A nonterminal SHIFTREDUCE chain can bypass reductions without C actions.
start ::= outer END .
outer ::= middle .
middle ::= inner .
inner ::= TOKEN .
