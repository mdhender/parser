// Package lalr parses PL/0 with the glcc-generated LALR(1) frontend.
package lalr

import "github.com/mdhender/parser/examples/pl0/lang"

//go:generate go run ../../../cmd/glcc -q -package lalr grammar.y

var kinds = map[string]int{
	"CONST": TK_CONST, "VAR": TK_VAR, "PROCEDURE": TK_PROCEDURE,
	"CALL": TK_CALL, "BEGIN": TK_BEGIN, "END": TK_END, "IF": TK_IF,
	"THEN": TK_THEN, "WHILE": TK_WHILE, "DO": TK_DO, "ODD": TK_ODD,
	"WRITE": TK_WRITE, "IDENT": TK_IDENT, "NUMBER": TK_NUMBER,
	"ASSIGN": TK_ASSIGN, "EQ": TK_EQ, "NE": TK_NE, "LT": TK_LT,
	"LE": TK_LE, "GT": TK_GT, "GE": TK_GE, "PLUS": TK_PLUS,
	"MINUS": TK_MINUS, "TIMES": TK_TIMES, "SLASH": TK_SLASH,
	"LPAREN": TK_LPAREN, "RPAREN": TK_RPAREN, "COMMA": TK_COMMA,
	"SEMI": TK_SEMI, "DOT": TK_DOT, "EOF": 0,
}

type syntaxError struct{ token lang.Token }

// Parse uses the shared lexer and returns the shared AST, never an executable
// action. The generated syntax hook aborts immediately without error recovery.
func Parse(source string) (block *lang.Block, err error) {
	tokens, err := lang.Lex(source)
	if err != nil {
		return nil, err
	}
	defer func() {
		if failure := recover(); failure != nil {
			syntax, ok := failure.(syntaxError)
			if !ok {
				panic(failure)
			}
			token := syntax.token
			if token.Line == 0 {
				token = tokens[len(tokens)-1]
			}
			block, err = nil, token.Errorf("unexpected %q", token.Text)
		}
	}()
	p := NewParser()
	defer p.Close()
	for _, token := range tokens {
		p.Parse(kinds[token.Kind], token, &block)
	}
	return block, nil
}
