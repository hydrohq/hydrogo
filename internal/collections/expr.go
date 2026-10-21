package collections

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// Expr is a small arithmetic expression over a record's number fields, used by
// computed fields: "quantity * unit_price * (1 + tax_rate)". It is evaluated
// with exact decimal arithmetic (big.Rat), never floats, and cannot call
// anything except the built-in functions round, abs, min and max.
//
//	expr    = term { ("+" | "-") term }
//	term    = unary { ("*" | "/") unary }
//	unary   = [ "-" ] primary
//	primary = number | field | func "(" expr { "," expr } ")" | "(" expr ")"
type Expr struct {
	root   exprNode
	fields []string
}

const (
	maxExprLength = 256
	maxExprNodes  = 64
	maxExprDepth  = 16
)

var ErrInvalidExpression = errors.New("invalid expression")

type exprNode interface {
	eval(vars func(string) *big.Rat) (*big.Rat, error)
}

type (
	numberNode struct{ value *big.Rat }
	fieldNode  struct{ name string }
	negNode    struct{ operand exprNode }
	binaryNode struct {
		op          byte
		left, right exprNode
	}
	callNode struct {
		name string
		args []exprNode
	}
)

// Fields lists the field names the expression reads.
func (e *Expr) Fields() []string { return append([]string(nil), e.fields...) }

// Eval computes the expression. vars returns a field's value (nil counts as
// 0). Division by zero is an error.
func (e *Expr) Eval(vars func(string) *big.Rat) (*big.Rat, error) { return e.root.eval(vars) }

func (n numberNode) eval(func(string) *big.Rat) (*big.Rat, error) { return n.value, nil }

func (n fieldNode) eval(vars func(string) *big.Rat) (*big.Rat, error) {
	if value := vars(n.name); value != nil {
		return value, nil
	}
	return new(big.Rat), nil
}

func (n negNode) eval(vars func(string) *big.Rat) (*big.Rat, error) {
	value, err := n.operand.eval(vars)
	if err != nil {
		return nil, err
	}
	return new(big.Rat).Neg(value), nil
}

func (n binaryNode) eval(vars func(string) *big.Rat) (*big.Rat, error) {
	left, err := n.left.eval(vars)
	if err != nil {
		return nil, err
	}
	right, err := n.right.eval(vars)
	if err != nil {
		return nil, err
	}
	out := new(big.Rat)
	switch n.op {
	case '+':
		return out.Add(left, right), nil
	case '-':
		return out.Sub(left, right), nil
	case '*':
		return out.Mul(left, right), nil
	default:
		if right.Sign() == 0 {
			return nil, errors.New("division by zero")
		}
		return out.Quo(left, right), nil
	}
}

func (n callNode) eval(vars func(string) *big.Rat) (*big.Rat, error) {
	values := make([]*big.Rat, len(n.args))
	for i, arg := range n.args {
		value, err := arg.eval(vars)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	switch n.name {
	case "abs":
		return new(big.Rat).Abs(values[0]), nil
	case "min":
		if values[0].Cmp(values[1]) <= 0 {
			return values[0], nil
		}
		return values[1], nil
	case "max":
		if values[0].Cmp(values[1]) >= 0 {
			return values[0], nil
		}
		return values[1], nil
	default: // round(x, places)
		places := values[1]
		if !places.IsInt() || places.Sign() < 0 || places.Num().Cmp(big.NewInt(MaxDecimals)) > 0 {
			return nil, errors.New("round needs 0 to 8 places")
		}
		return RoundHalfUp(values[0], int(places.Num().Int64())), nil
	}
}

// RoundHalfUp rounds x to places decimal places, halves away from zero.
func RoundHalfUp(x *big.Rat, places int) *big.Rat {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	scaled := new(big.Rat).Mul(x, new(big.Rat).SetInt(scale))
	negative := scaled.Sign() < 0
	scaled.Abs(scaled)
	// floor(scaled + 1/2)
	scaled.Add(scaled, big.NewRat(1, 2))
	whole := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	if negative {
		whole.Neg(whole)
	}
	return new(big.Rat).SetFrac(whole, scale)
}

var exprFunctions = map[string]int{"abs": 1, "min": 2, "max": 2, "round": 2}

// ParseExpr parses an expression. The result is safe to evaluate for any
// field values: its size and nesting are bounded.
func ParseExpr(source string) (*Expr, error) {
	if strings.TrimSpace(source) == "" || len(source) > maxExprLength {
		return nil, ErrInvalidExpression
	}
	p := &exprParser{src: source}
	root, err := p.expr(0)
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.pos != len(p.src) {
		return nil, fmt.Errorf("%w: unexpected %q", ErrInvalidExpression, p.src[p.pos:p.pos+1])
	}
	return &Expr{root: root, fields: p.fields}, nil
}

type exprParser struct {
	src    string
	pos    int
	nodes  int
	fields []string
}

func (p *exprParser) skip() {
	for p.pos < len(p.src) && p.src[p.pos] == ' ' {
		p.pos++
	}
}

func (p *exprParser) peek() byte {
	p.skip()
	if p.pos < len(p.src) {
		return p.src[p.pos]
	}
	return 0
}

func (p *exprParser) node() error {
	p.nodes++
	if p.nodes > maxExprNodes {
		return fmt.Errorf("%w: too long", ErrInvalidExpression)
	}
	return nil
}

func (p *exprParser) expr(depth int) (exprNode, error) {
	if depth > maxExprDepth {
		return nil, fmt.Errorf("%w: too deep", ErrInvalidExpression)
	}
	left, err := p.term(depth)
	if err != nil {
		return nil, err
	}
	for c := p.peek(); c == '+' || c == '-'; c = p.peek() {
		p.pos++
		right, err := p.term(depth)
		if err != nil {
			return nil, err
		}
		if err := p.node(); err != nil {
			return nil, err
		}
		left = binaryNode{op: c, left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) term(depth int) (exprNode, error) {
	left, err := p.unary(depth)
	if err != nil {
		return nil, err
	}
	for c := p.peek(); c == '*' || c == '/'; c = p.peek() {
		p.pos++
		right, err := p.unary(depth)
		if err != nil {
			return nil, err
		}
		if err := p.node(); err != nil {
			return nil, err
		}
		left = binaryNode{op: c, left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) unary(depth int) (exprNode, error) {
	if p.peek() == '-' {
		p.pos++
		operand, err := p.unary(depth + 1)
		if err != nil {
			return nil, err
		}
		if err := p.node(); err != nil {
			return nil, err
		}
		return negNode{operand}, nil
	}
	return p.primary(depth)
}

func (p *exprParser) primary(depth int) (exprNode, error) {
	c := p.peek()
	if err := p.node(); err != nil {
		return nil, err
	}
	switch {
	case c == '(':
		p.pos++
		inner, err := p.expr(depth + 1)
		if err != nil {
			return nil, err
		}
		if p.peek() != ')' {
			return nil, fmt.Errorf("%w: missing )", ErrInvalidExpression)
		}
		p.pos++
		return inner, nil
	case c >= '0' && c <= '9' || c == '.':
		start := p.pos
		for p.pos < len(p.src) && (p.src[p.pos] >= '0' && p.src[p.pos] <= '9' || p.src[p.pos] == '.') {
			p.pos++
		}
		value, ok := new(big.Rat).SetString(p.src[start:p.pos])
		if !ok {
			return nil, fmt.Errorf("%w: bad number %q", ErrInvalidExpression, p.src[start:p.pos])
		}
		return numberNode{value}, nil
	case c >= 'a' && c <= 'z':
		start := p.pos
		for p.pos < len(p.src) && (p.src[p.pos] >= 'a' && p.src[p.pos] <= 'z' || p.src[p.pos] >= '0' && p.src[p.pos] <= '9' || p.src[p.pos] == '_') {
			p.pos++
		}
		name := p.src[start:p.pos]
		if p.peek() != '(' {
			p.fields = append(p.fields, name)
			return fieldNode{name}, nil
		}
		arity, known := exprFunctions[name]
		if !known {
			return nil, fmt.Errorf("%w: unknown function %q", ErrInvalidExpression, name)
		}
		p.pos++
		var args []exprNode
		for {
			arg, err := p.expr(depth + 1)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.peek() == ',' {
				p.pos++
				continue
			}
			break
		}
		if p.peek() != ')' || len(args) != arity {
			return nil, fmt.Errorf("%w: %s takes %d arguments", ErrInvalidExpression, name, arity)
		}
		p.pos++
		return callNode{name: name, args: args}, nil
	default:
		return nil, fmt.Errorf("%w: unexpected input at position %d", ErrInvalidExpression, p.pos)
	}
}
