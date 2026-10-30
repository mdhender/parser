// The state after PREFIX can shift ANY and reduce the empty optional.
// Its wildcard shift must prevent choosing that reduction as a default.
%wildcard ANY .
start ::= PREFIX optional END .
optional ::= .
optional ::= ANY .
