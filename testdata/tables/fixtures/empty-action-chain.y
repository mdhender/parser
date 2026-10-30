// Explicit empty actions still block the noCode SHIFTREDUCE optimization.
start ::= outer END .
outer ::= middle . {}
middle ::= inner . {}
inner ::= TOKEN . {}
