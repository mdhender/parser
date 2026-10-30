%destructor list { first($$); }
%name "CaféParser"
%destructor list { second($$); }
%destructor list {
  /* UTF-8 bytes: café, 日本語, λ. Braces: { } */
  third($$);
}
%destructor ITEM { token_first($$); }
%destructor ITEM { token_second($$); }
%type list {struct Café *}
%token_prefix "préfixe_"
start ::= list .
list(value) ::= ITEM(token) . { value = café(token); }
