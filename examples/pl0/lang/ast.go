// Package lang defines the shared PL/0 syntax tree, lexer, and interpreter.
// Neither parser frontend is a dependency of this package.
package lang

import "fmt"

// Token retains source spelling and a one-based byte position. Kind is a
// symbolic terminal name, independent of either generator's numeric token IDs.
type Token struct {
	Kind, Text   string
	Number       int64
	Line, Column int
}

func (t Token) Errorf(format string, args ...any) error {
	return fmt.Errorf("%d:%d: %s", t.Line, t.Column, fmt.Sprintf(format, args...))
}

type Block struct {
	Constants  []Constant
	Variables  []Token
	Procedures []Procedure
	Statement  *Statement
}

type Constant struct {
	Name, Value Token
}

type Procedure struct {
	Name  Token
	Block *Block
}

// Expr is a number/name leaf, unary expression (Left nil), or binary
// expression. Token identifies the literal, name, or operator respectively.
type Expr struct {
	Token       Token
	Left, Right *Expr
}

// Condition has one operand for ODD and two for a comparison.
type Condition struct {
	Token       Token
	Left, Right *Expr
}

// Statement.Kind is assign, call, begin, if, while, or write. Empty statements
// are nil. Body holds the compound statements or the single if/while body.
type Statement struct {
	Kind      string
	Token     Token
	Expr      *Expr
	Condition *Condition
	Body      []*Statement
}
