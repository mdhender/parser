%name ""
%include {}
%code {}
%token_type {}
%default_type {}
%type start {}
%destructor start {}
start(alias) ::= TOKEN(value) . {}
