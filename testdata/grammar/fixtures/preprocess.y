%name Conditional
%ifdef ALPHA
%include { alpha(); }
%else
%include { no_alpha(); }
%endif
%ifndef BETA
%token EXTRA .
%endif
%if ALPHA && (!BETA || GAMMA)
start ::= YES .
%else
start ::= NO .
%endif
%ifdef OUTER
%ifdef INNER
start ::= NESTED .
%else
start ::= OUTER_ONLY .
%endif
%endif
%if ALPHA || UNREACHED
start ::= SHORTCIRCUIT .
%endif
start ::= BASE .
