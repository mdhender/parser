# Why glime uses deferred completion

LALR Lemon selects reductions using states and lookahead. glime instead chooses
a production before parsing its children, using the current nonterminal and one
lookahead token. SELECT consists of FIRST of the RHS, plus FOLLOW of the LHS
when the RHS can be empty. Each table cell must identify at most one rule.
FOLLOW starts with EOF for the start symbol and propagates across nullable
suffixes. The fixed-point analysis is independent of Lemon's LR automaton.

Choosing a production is too early to execute its semantic action: its children
have no values yet. The default stack strategy allocates a frame for that production,
pushes its completion marker, then pushes the RHS in reverse order. Terminals
fill frame slots as they match; completed nonterminals fill their parent's slot.
Only when a marker reaches the top does its action read the complete RHS and
return the LHS value. An empty production reaches its marker immediately.

Each expansion owns independent storage, so recursive calls cannot overwrite
their ancestors' values. The stack runtime uses an explicit growing stack, not Go
recursion for grammar traversal. User actions can still introduce Go recursion,
for example the calculator's continuation functions.

The grammar frontend is shared with Lemon through a parse-only entry point.
glime never constructs LR states, propagates LR lookaheads, resolves shift/reduce
conflicts, or packs LR tables. Its simple predictive table favors auditability
over compression.

The recursive-descent strategy uses exactly the same predictive decisions, but
emits a function for each nonterminal. Each function switches on lookahead,
matches terminals and calls child functions in RHS order, then runs the action
after all children return successfully. Epsilon productions complete immediately
when predicted by FOLLOW, including implicit EOF. Each invocation owns its own
semantic slots, just as each explicit-stack expansion does. Thus completed child
actions remain observable if a later child or trailing input fails.

Recursive descent makes grammar traversal visible as ordinary Go calls, at the
cost of call-stack depth for both nesting and right-recursive sequences. The
stack strategy instead stores pending work on a heap-backed slice. Neither
choice changes which grammars are eligible, and neither automatically makes
user actions iterative or bounds their resource use.
