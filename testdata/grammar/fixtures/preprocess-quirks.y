// Lemon short-circuits the remaining expression, not C-precedence subterms.
%if ABSENT && ALPHA || ALPHA
start ::= C_PRECEDENCE_WOULD_INCLUDE .
%else
start ::= WHOLE_EXPRESSION_SHORTCIRCUIT .
%endif
%if (ABSENT && ALPHA) || ALPHA
start ::= PARENTHESES_RESTORE_EVALUATION .
%endif
%if ALPHA || @ ignored malformed tail
start ::= SHORTCIRCUIT_SKIPS_SYNTAX .
%endif
%ifdef ABSENT
%if @ excluded malformed expression is not evaluated
this is not even a grammar;
%endif
%endif
%include {
  %if ALPHA
  int indented_directive_is_preserved;
  %endif
}
// An active conditional missing its closing endif is accepted upstream.
%ifdef ALPHA
start ::= ACTIVE_WITHOUT_ENDIF .
