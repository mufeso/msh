package expr

import (
	"fmt"
	"regexp"
)

// node is an element of the syntax tree.
type node interface{}

type (
	identNode   struct{ name string }
	literalNode struct{ value string }
	notNode     struct{ inner node }
	andNode     struct{ left, right node }
	orNode      struct{ left, right node }
	compareNode struct {
		op          string
		left, right node
	}
	matchNode struct {
		left   node
		re     *regexp.Regexp
		negate bool
	}
)

// Expr is a parsed expression.
type Expr struct {
	src  string
	root node
}

// String returns the source text.
func (e *Expr) String() string { return e.src }

// Parse compiles an expression.
//
//	or      = and { "||" and }
//	and     = unary { "&&" unary }
//	unary   = "!" unary | compare
//	compare = primary [ op primary ]
//	primary = ident | string | number | "(" or ")"
func Parse(src string) (*Expr, error) {
	tokens, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	root, err := p.or()
	if err != nil {
		return nil, err
	}
	if t := p.peek(); t.kind != tokEOF {
		return nil, fmt.Errorf("position %d: unexpected %q", t.pos+1, t.text)
	}
	return &Expr{src: src, root: root}, nil
}

// maxDepth limits nesting of parentheses and "!". Each level is a function
// call, so unlimited depth would let hostile input overflow the stack.
const maxDepth = 64

type parser struct {
	tokens []token
	pos    int
	depth  int
}

// enter counts one more nesting level.
func (p *parser) enter() error {
	p.depth++
	if p.depth > maxDepth {
		return fmt.Errorf("position %d: expression nested too deeply", p.peek().pos+1)
	}
	return nil
}

func (p *parser) peek() token { return p.tokens[p.pos] }

func (p *parser) next() token {
	t := p.tokens[p.pos]
	if t.kind != tokEOF {
		p.pos++
	}
	return t
}

func (p *parser) or() (node, error) {
	left, err := p.and()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOr {
		p.next()
		right, err := p.and()
		if err != nil {
			return nil, err
		}
		left = orNode{left, right}
	}
	return left, nil
}

func (p *parser) and() (node, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokAnd {
		p.next()
		right, err := p.unary()
		if err != nil {
			return nil, err
		}
		left = andNode{left, right}
	}
	return left, nil
}

func (p *parser) unary() (node, error) {
	if p.peek().kind == tokNot {
		p.next()
		if err := p.enter(); err != nil {
			return nil, err
		}
		defer func() { p.depth-- }()
		inner, err := p.unary()
		if err != nil {
			return nil, err
		}
		return notNode{inner}, nil
	}
	return p.compare()
}

func (p *parser) compare() (node, error) {
	left, err := p.primary()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokOp {
		return left, nil
	}
	op := p.next()
	if op.text == "~" || op.text == "!~" {
		pattern := p.next()
		if pattern.kind != tokString {
			return nil, fmt.Errorf("position %d: %s needs a quoted regular expression", op.pos+1, op.text)
		}
		re, err := regexp.Compile(pattern.text)
		if err != nil {
			return nil, fmt.Errorf("position %d: %v", pattern.pos+1, err)
		}
		return matchNode{left: left, re: re, negate: op.text == "!~"}, nil
	}
	right, err := p.primary()
	if err != nil {
		return nil, err
	}
	return compareNode{op: op.text, left: left, right: right}, nil
}

func (p *parser) primary() (node, error) {
	t := p.next()
	switch t.kind {
	case tokIdent:
		return identNode{t.text}, nil
	case tokString, tokNumber:
		return literalNode{t.text}, nil
	case tokLParen:
		if err := p.enter(); err != nil {
			return nil, err
		}
		defer func() { p.depth-- }()
		inner, err := p.or()
		if err != nil {
			return nil, err
		}
		if closing := p.next(); closing.kind != tokRParen {
			return nil, fmt.Errorf("position %d: missing )", closing.pos+1)
		}
		return inner, nil
	case tokEOF:
		return nil, fmt.Errorf("position %d: unexpected end of expression", t.pos+1)
	default:
		return nil, fmt.Errorf("position %d: unexpected %q", t.pos+1, t.text)
	}
}
