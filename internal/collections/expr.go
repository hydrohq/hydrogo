package collections

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// Expr is a small, side-effect free expression over a record's fields, used by
// computed fields ("quantity * unit_price"), rules ("total > 0 && status !=
// 'void'") and state transitions. Numbers use exact decimal arithmetic
// (big.Rat), never floats. An expression can call nothing except the built-in
// functions abs, min, max, round and empty.
//
//	or      = and { "||" and }
//	and     = not { "&&" not }
//	not     = "!" not | compare
//	compare = sum [ ("==" | "!=" | "<" | "<=" | ">" | ">=") sum ]
//	sum     = term { ("+" | "-") term }
//	term    = unary { ("*" | "/") unary }
//	unary   = [ "-" ] primary
//	primary = number | 'text' | true | false | field | func "(" args ")" | "(" or ")"
//
// A value is nil (field not set), a bool, a string or a number (*big.Rat). In
// arithmetic a missing value is 0; in comparisons it equals "", 0 and false.
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

// Value is what an expression evaluates to.
type Value = any

type exprNode interface {
	eval(vars func(string) Value) (Value, error)
}

type (
	literalNode struct{ value Value }
	fieldNode   struct{ name string }
	negNode     struct{ operand exprNode }
	notNode     struct{ operand exprNode }
	binaryNode  struct {
		op          string
		left, right exprNode
	}
	callNode struct {
		name string
		args []exprNode
	}
)

// Fields lists the field names the expression reads.
func (e *Expr) Fields() []string { return append([]string(nil), e.fields...) }

// Eval computes the expression. vars returns a field's value: nil, bool,
// string or *big.Rat.
func (e *Expr) Eval(vars func(string) Value) (Value, error) { return e.root.eval(vars) }

// EvalBool evaluates a condition; anything but a bool is an error.
func (e *Expr) EvalBool(vars func(string) Value) (bool, error) {
	value, err := e.root.eval(vars)
	if err != nil {
		return false, err
	}
	result, ok := value.(bool)
	if !ok {
		return false, errors.New("condition is not true or false")
	}
	return result, nil
}

// EvalNumber evaluates a numeric expression.
func (e *Expr) EvalNumber(vars func(string) Value) (*big.Rat, error) {
	value, err := e.root.eval(vars)
	if err != nil {
		return nil, err
	}
	return asNumber(value)
}

var errNotNumber = errors.New("expected a number")

func asNumber(value Value) (*big.Rat, error) {
	switch typed := value.(type) {
	case *big.Rat:
		return typed, nil
	case nil:
		return new(big.Rat), nil
	default:
		return nil, errNotNumber
	}
}

func (n literalNode) eval(func(string) Value) (Value, error) { return n.value, nil }

func (n fieldNode) eval(vars func(string) Value) (Value, error) { return vars(n.name), nil }

func (n negNode) eval(vars func(string) Value) (Value, error) {
	value, err := n.operand.eval(vars)
	if err != nil {
		return nil, err
	}
	number, err := asNumber(value)
	if err != nil {
		return nil, err
	}
	return new(big.Rat).Neg(number), nil
}

func (n notNode) eval(vars func(string) Value) (Value, error) {
	value, err := n.operand.eval(vars)
	if err != nil {
		return nil, err
	}
	flag, ok := value.(bool)
	if !ok {
		return nil, errors.New("! needs a true or false value")
	}
	return !flag, nil
}

func (n binaryNode) eval(vars func(string) Value) (Value, error) {
	left, err := n.left.eval(vars)
	if err != nil {
		return nil, err
	}
	// && and || only evaluate the right side when needed.
	if n.op == "&&" || n.op == "||" {
		l, ok := left.(bool)
		if !ok {
			return nil, errors.New(n.op + " needs true or false values")
		}
		if (n.op == "&&" && !l) || (n.op == "||" && l) {
			return l, nil
		}
		right, err := n.right.eval(vars)
		if err != nil {
			return nil, err
		}
		r, ok := right.(bool)
		if !ok {
			return nil, errors.New(n.op + " needs true or false values")
		}
		return r, nil
	}
	right, err := n.right.eval(vars)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "==", "!=", "<", "<=", ">", ">=":
		return compareValues(n.op, left, right)
	}
	l, err := asNumber(left)
	if err != nil {
		return nil, err
	}
	r, err := asNumber(right)
	if err != nil {
		return nil, err
	}
	out := new(big.Rat)
	switch n.op {
	case "+":
		return out.Add(l, r), nil
	case "-":
		return out.Sub(l, r), nil
	case "*":
		return out.Mul(l, r), nil
	default:
		if r.Sign() == 0 {
			return nil, errors.New("division by zero")
		}
		return out.Quo(l, r), nil
	}
}

// compareValues compares two values of the same kind; nil takes the zero value
// of the other side.
func compareValues(op string, left, right Value) (Value, error) {
	if left == nil {
		left = zeroLike(right)
	}
	if right == nil {
		right = zeroLike(left)
	}
	var order int
	switch l := left.(type) {
	case *big.Rat:
		r, ok := right.(*big.Rat)
		if !ok {
			return nil, errors.New("cannot compare a number with another kind of value")
		}
		order = l.Cmp(r)
	case string:
		r, ok := right.(string)
		if !ok {
			return nil, errors.New("cannot compare text with another kind of value")
		}
		order = strings.Compare(l, r)
	case bool:
		r, ok := right.(bool)
		if !ok || (op != "==" && op != "!=") {
			return nil, errors.New("true or false values can only be compared with == and !=")
		}
		if l != r {
			order = 1
		}
	default:
		order = 0
	}
	switch op {
	case "==":
		return order == 0, nil
	case "!=":
		return order != 0, nil
	case "<":
		return order < 0, nil
	case "<=":
		return order <= 0, nil
	case ">":
		return order > 0, nil
	default:
		return order >= 0, nil
	}
}

func zeroLike(other Value) Value {
	switch other.(type) {
	case *big.Rat:
		return new(big.Rat)
	case string:
		return ""
	case bool:
		return false
	default:
		return nil
	}
}

func (n callNode) eval(vars func(string) Value) (Value, error) {
	values := make([]Value, len(n.args))
	for i, arg := range n.args {
		value, err := arg.eval(vars)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	if n.name == "empty" {
		switch typed := values[0].(type) {
		case nil:
			return true, nil
		case string:
			return typed == "", nil
		default:
			return false, nil
		}
	}
	numbers := make([]*big.Rat, len(values))
	for i, value := range values {
		number, err := asNumber(value)
		if err != nil {
			return nil, err
		}
		numbers[i] = number
	}
	switch n.name {
	case "abs":
		return new(big.Rat).Abs(numbers[0]), nil
	case "min":
		if numbers[0].Cmp(numbers[1]) <= 0 {
			return numbers[0], nil
		}
		return numbers[1], nil
	case "max":
		if numbers[0].Cmp(numbers[1]) >= 0 {
			return numbers[0], nil
		}
		return numbers[1], nil
	default: // round(x, places)
		places := numbers[1]
		if !places.IsInt() || places.Sign() < 0 || places.Num().Cmp(big.NewInt(MaxDecimals)) > 0 {
			return nil, errors.New("round needs 0 to 8 places")
		}
		return RoundHalfUp(numbers[0], int(places.Num().Int64())), nil
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

var exprFunctions = map[string]int{"abs": 1, "min": 2, "max": 2, "round": 2, "empty": 1}

// ParseExpr parses an expression. The result is safe to evaluate for any
// field values: its size and nesting are bounded.
func ParseExpr(source string) (*Expr, error) {
	if strings.TrimSpace(source) == "" || len(source) > maxExprLength {
		return nil, ErrInvalidExpression
	}
	p := &exprParser{src: source}
	root, err := p.or(0)
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

// take consumes op when the input continues with it.
func (p *exprParser) take(op string) bool {
	p.skip()
	if strings.HasPrefix(p.src[p.pos:], op) {
		p.pos += len(op)
		return true
	}
	return false
}

func (p *exprParser) node() error {
	p.nodes++
	if p.nodes > maxExprNodes {
		return fmt.Errorf("%w: too long", ErrInvalidExpression)
	}
	return nil
}

func (p *exprParser) binary(op string, left, right exprNode) (exprNode, error) {
	if err := p.node(); err != nil {
		return nil, err
	}
	return binaryNode{op: op, left: left, right: right}, nil
}

func (p *exprParser) or(depth int) (exprNode, error) {
	if depth > maxExprDepth {
		return nil, fmt.Errorf("%w: too deep", ErrInvalidExpression)
	}
	left, err := p.and(depth)
	if err != nil {
		return nil, err
	}
	for p.take("||") {
		right, err := p.and(depth)
		if err != nil {
			return nil, err
		}
		if left, err = p.binary("||", left, right); err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *exprParser) and(depth int) (exprNode, error) {
	left, err := p.not(depth)
	if err != nil {
		return nil, err
	}
	for p.take("&&") {
		right, err := p.not(depth)
		if err != nil {
			return nil, err
		}
		if left, err = p.binary("&&", left, right); err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *exprParser) not(depth int) (exprNode, error) {
	p.skip()
	if p.pos < len(p.src) && p.src[p.pos] == '!' && !strings.HasPrefix(p.src[p.pos:], "!=") {
		p.pos++
		if depth > maxExprDepth {
			return nil, fmt.Errorf("%w: too deep", ErrInvalidExpression)
		}
		operand, err := p.not(depth + 1)
		if err != nil {
			return nil, err
		}
		if err := p.node(); err != nil {
			return nil, err
		}
		return notNode{operand}, nil
	}
	return p.compare(depth)
}

func (p *exprParser) compare(depth int) (exprNode, error) {
	left, err := p.sum(depth)
	if err != nil {
		return nil, err
	}
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if p.take(op) {
			right, err := p.sum(depth)
			if err != nil {
				return nil, err
			}
			return p.binary(op, left, right)
		}
	}
	return left, nil
}

func (p *exprParser) sum(depth int) (exprNode, error) {
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
		if left, err = p.binary(string(c), left, right); err != nil {
			return nil, err
		}
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
		if left, err = p.binary(string(c), left, right); err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *exprParser) unary(depth int) (exprNode, error) {
	if p.peek() == '-' {
		p.pos++
		if depth > maxExprDepth {
			return nil, fmt.Errorf("%w: too deep", ErrInvalidExpression)
		}
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
		inner, err := p.or(depth + 1)
		if err != nil {
			return nil, err
		}
		if p.peek() != ')' {
			return nil, fmt.Errorf("%w: missing )", ErrInvalidExpression)
		}
		p.pos++
		return inner, nil
	case c == '\'':
		end := strings.IndexByte(p.src[p.pos+1:], '\'')
		if end < 0 {
			return nil, fmt.Errorf("%w: unterminated text", ErrInvalidExpression)
		}
		text := p.src[p.pos+1 : p.pos+1+end]
		p.pos += end + 2
		return literalNode{text}, nil
	case c >= '0' && c <= '9' || c == '.':
		start := p.pos
		for p.pos < len(p.src) && (p.src[p.pos] >= '0' && p.src[p.pos] <= '9' || p.src[p.pos] == '.') {
			p.pos++
		}
		value, ok := new(big.Rat).SetString(p.src[start:p.pos])
		if !ok {
			return nil, fmt.Errorf("%w: bad number %q", ErrInvalidExpression, p.src[start:p.pos])
		}
		return literalNode{value}, nil
	case c >= 'a' && c <= 'z':
		start := p.pos
		for p.pos < len(p.src) && (p.src[p.pos] >= 'a' && p.src[p.pos] <= 'z' || p.src[p.pos] >= '0' && p.src[p.pos] <= '9' || p.src[p.pos] == '_') {
			p.pos++
		}
		name := p.src[start:p.pos]
		if p.peek() != '(' {
			switch name {
			case "true":
				return literalNode{true}, nil
			case "false":
				return literalNode{false}, nil
			}
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
			arg, err := p.or(depth + 1)
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
