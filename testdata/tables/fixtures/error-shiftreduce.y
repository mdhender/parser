// error must retain SHIFTREDUCE's shift component, unlike ordinary symbols.
// Explicit actions prevent noCode bypass from hiding either comparison.
start ::= item END .
item ::= error . {}
item ::= ordinary . {}
ordinary ::= TOKEN . {}
