package lang

import (
	"fmt"
	"io"
)

// Execute validates every block before executing the program. Procedures in a
// block are mutually visible, including forward references; each call gets fresh
// variables and retains its declaration's lexical environment.
func Execute(block *Block, output io.Writer) error {
	scope, err := checkBlock(block, nil)
	if err != nil {
		return err
	}
	frame := &activation{scope: scope, variables: make(map[string]int64)}
	return frame.execute(block.Statement, output)
}

type declaration struct {
	kind      string
	value     int64
	procedure *lexicalScope
}

type lexicalScope struct {
	parent *lexicalScope
	block  *Block
	names  map[string]declaration
}

func (s *lexicalScope) declare(name Token, d declaration) error {
	if _, exists := s.names[name.Text]; exists {
		return name.Errorf("duplicate declaration of %q", name.Text)
	}
	s.names[name.Text] = d
	return nil
}

func (s *lexicalScope) lookup(name Token) (declaration, error) {
	for current := s; current != nil; current = current.parent {
		if d, ok := current.names[name.Text]; ok {
			return d, nil
		}
	}
	return declaration{}, name.Errorf("undefined name %q", name.Text)
}

func checkBlock(block *Block, parent *lexicalScope) (*lexicalScope, error) {
	scope := &lexicalScope{parent: parent, block: block, names: make(map[string]declaration)}
	for _, c := range block.Constants {
		if err := scope.declare(c.Name, declaration{kind: "constant", value: c.Value.Number}); err != nil {
			return nil, err
		}
	}
	for _, name := range block.Variables {
		if err := scope.declare(name, declaration{kind: "variable"}); err != nil {
			return nil, err
		}
	}
	// Register all procedure names before checking any body, so siblings may
	// recurse mutually, independent of their source order.
	for _, p := range block.Procedures {
		if err := scope.declare(p.Name, declaration{kind: "procedure"}); err != nil {
			return nil, err
		}
	}
	for _, p := range block.Procedures {
		body, err := checkBlock(p.Block, scope)
		if err != nil {
			return nil, err
		}
		scope.names[p.Name.Text] = declaration{kind: "procedure", procedure: body}
	}
	if err := scope.checkStatement(block.Statement); err != nil {
		return nil, err
	}
	return scope, nil
}

func (s *lexicalScope) require(name Token, kind string) error {
	d, err := s.lookup(name)
	if err != nil {
		return err
	}
	if d.kind != kind {
		return name.Errorf("%q is a %s, not a %s", name.Text, d.kind, kind)
	}
	return nil
}

func (s *lexicalScope) checkExpr(expr *Expr) error {
	switch expr.Token.Kind {
	case "NUMBER":
		return nil
	case "IDENT":
		d, err := s.lookup(expr.Token)
		if err != nil {
			return err
		}
		if d.kind == "procedure" {
			return expr.Token.Errorf("%q is a procedure, not a value", expr.Token.Text)
		}
		return nil
	}
	if expr.Left != nil {
		if err := s.checkExpr(expr.Left); err != nil {
			return err
		}
	}
	return s.checkExpr(expr.Right)
}

func (s *lexicalScope) checkStatement(stmt *Statement) error {
	if stmt == nil {
		return nil
	}
	switch stmt.Kind {
	case "assign":
		if err := s.require(stmt.Token, "variable"); err != nil {
			return err
		}
		return s.checkExpr(stmt.Expr)
	case "call":
		return s.require(stmt.Token, "procedure")
	case "write":
		return s.checkExpr(stmt.Expr)
	case "if", "while":
		if err := s.checkExpr(stmt.Condition.Left); err != nil {
			return err
		}
		if stmt.Condition.Token.Kind != "ODD" {
			if err := s.checkExpr(stmt.Condition.Right); err != nil {
				return err
			}
		}
	}
	for _, child := range stmt.Body {
		if err := s.checkStatement(child); err != nil {
			return err
		}
	}
	return nil
}

type activation struct {
	scope     *lexicalScope
	parent    *activation
	variables map[string]int64
}

func (a *activation) lookup(name string) (declaration, *activation) {
	for current := a; current != nil; current = current.parent {
		if d, ok := current.scope.names[name]; ok {
			return d, current
		}
	}
	panic("name not validated")
}

func (a *activation) eval(expr *Expr) (int64, error) {
	switch expr.Token.Kind {
	case "NUMBER":
		return expr.Token.Number, nil
	case "IDENT":
		d, owner := a.lookup(expr.Token.Text)
		if d.kind == "constant" {
			return d.value, nil
		}
		return owner.variables[expr.Token.Text], nil
	}
	var left int64
	if expr.Left != nil {
		var err error
		left, err = a.eval(expr.Left)
		if err != nil {
			return 0, err
		}
	}
	right, err := a.eval(expr.Right)
	if err != nil {
		return 0, err
	}
	switch expr.Token.Kind {
	case "PLUS":
		return left + right, nil
	case "MINUS":
		return left - right, nil
	case "TIMES":
		return left * right, nil
	case "SLASH":
		if right == 0 {
			return 0, expr.Token.Errorf("division by zero")
		}
		return left / right, nil
	}
	panic("unknown expression operator")
}

func (a *activation) test(condition *Condition) (bool, error) {
	left, err := a.eval(condition.Left)
	if err != nil {
		return false, err
	}
	if condition.Token.Kind == "ODD" {
		return left%2 != 0, nil
	}
	right, err := a.eval(condition.Right)
	if err != nil {
		return false, err
	}
	switch condition.Token.Kind {
	case "EQ":
		return left == right, nil
	case "NE":
		return left != right, nil
	case "LT":
		return left < right, nil
	case "LE":
		return left <= right, nil
	case "GT":
		return left > right, nil
	case "GE":
		return left >= right, nil
	}
	panic("unknown condition operator")
}

func (a *activation) execute(stmt *Statement, output io.Writer) error {
	if stmt == nil {
		return nil
	}
	switch stmt.Kind {
	case "assign":
		value, err := a.eval(stmt.Expr)
		if err != nil {
			return err
		}
		_, owner := a.lookup(stmt.Token.Text)
		owner.variables[stmt.Token.Text] = value
	case "call":
		d, owner := a.lookup(stmt.Token.Text)
		callee := &activation{scope: d.procedure, parent: owner, variables: make(map[string]int64)}
		return callee.execute(d.procedure.block.Statement, output)
	case "write":
		value, err := a.eval(stmt.Expr)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, value)
		return err
	case "begin":
		for _, child := range stmt.Body {
			if err := a.execute(child, output); err != nil {
				return err
			}
		}
	case "if", "while":
		for {
			ok, err := a.test(stmt.Condition)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			for _, child := range stmt.Body {
				if err := a.execute(child, output); err != nil {
					return err
				}
			}
			if stmt.Kind == "if" {
				return nil
			}
		}
	}
	return nil
}
