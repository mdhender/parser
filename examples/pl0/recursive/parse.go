// Package recursive parses PL/0 with glime's recursive-descent frontend.
package recursive

import "github.com/mdhender/parser/examples/pl0/lang"

// Reuse the stack frontend's grammar; a separate package keeps generated names
// and token IDs private to each frontend without rewriting grammar declarations.
//go:generate go run ../../../cmd/glime -strategy=recursive -package recursive -d . generate ../ll1/grammar.y

var kinds = map[string]int{
	"CONST": TK_CONST, "VAR": TK_VAR, "PROCEDURE": TK_PROCEDURE,
	"CALL": TK_CALL, "BEGIN": TK_BEGIN, "END": TK_END, "IF": TK_IF,
	"THEN": TK_THEN, "WHILE": TK_WHILE, "DO": TK_DO, "ODD": TK_ODD,
	"WRITE": TK_WRITE, "IDENT": TK_IDENT, "NUMBER": TK_NUMBER,
	"ASSIGN": TK_ASSIGN, "EQ": TK_EQ, "NE": TK_NE, "LT": TK_LT,
	"LE": TK_LE, "GT": TK_GT, "GE": TK_GE, "PLUS": TK_PLUS,
	"MINUS": TK_MINUS, "TIMES": TK_TIMES, "SLASH": TK_SLASH,
	"LPAREN": TK_LPAREN, "RPAREN": TK_RPAREN, "COMMA": TK_COMMA,
	"SEMI": TK_SEMI, "DOT": TK_DOT,
}

type syntaxError struct{ token lang.Token }

func fail(token lang.Token) { panic(syntaxError{token}) }

// operation and fold implement private LL(1) action bookkeeping. Expression
// tails become left-associative trees, never additional shared AST node types.
type operation struct {
	token lang.Token
	right *lang.Expr
}

func fold(left *lang.Expr, tail []operation) *lang.Expr {
	for _, op := range tail {
		left = &lang.Expr{Token: op.token, Left: left, Right: op.right}
	}
	return left
}

// Parse adapts the shared lexer's tokens to glime's complete-input API.
// Only the syntax hook's typed panic is caught, preserving source locations;
// the implicit EOF token is recovered from the shared lexer's final token.
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
	input := make([]ParseTokensToken, 0, len(tokens)-1)
	for _, token := range tokens[:len(tokens)-1] {
		input = append(input, ParseTokensToken{Kind: kinds[token.Kind], Value: token})
	}
	return ParseTokens(input)
}
