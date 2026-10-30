%left PLUS .
%right POWER .
%nonassoc EQ .
start ::= expr .
expr ::= expr PLUS expr .
expr ::= expr POWER expr .
expr ::= expr EQ expr .
expr ::= ID .
