package sqlast

import (
	"fmt"
	"strings"
)

type binaryOperator int

const (
	binaryOpUnknown binaryOperator = iota
	binaryOpOR
	binaryOpAND
)

type binaryOp struct {
	source RegoSource
	op     binaryOperator

	Terms []BooleanNode
}

func (binaryOp) UseAs() Node    { return binaryOp{} }
func (binaryOp) IsBooleanNode() {}

func Or(source RegoSource, terms ...BooleanNode) BooleanNode {
	return newBinaryOp(source, binaryOpOR, terms...)
}

func And(source RegoSource, terms ...BooleanNode) BooleanNode {
	return newBinaryOp(source, binaryOpAND, terms...)
}

func newBinaryOp(source RegoSource, op binaryOperator, terms ...BooleanNode) BooleanNode {
	if len(terms) == 0 {
		// TODO: How to handle 0 terms?
		return Bool(false)
	}

	if len(terms) == 1 {
		return terms[0]
	}

	return binaryOp{
		Terms:  terms,
		op:     op,
		source: source,
	}
}

func (b binaryOp) SQLString(cfg *SQLGenerator) string {
	sqlOp := ""
	switch b.op {
	case binaryOpOR:
		sqlOp = "OR"
	case binaryOpAND:
		sqlOp = "AND"
	default:
		cfg.AddError(fmt.Errorf("unsupported binary operator: %s (%d)", b.source, b.op))
		return "BinaryOpError"
	}

	terms := make([]string, 0, len(b.Terms))
	for _, term := range b.Terms {
		termSql := term.SQLString(cfg)

		switch term.UseAs().(type) {
		case AstBoolean:
		default:
			// By default, wrap all terms in parens. This might be excessive
			// but it is safe.
			termSql = "(" + termSql + ")"
		}

		terms = append(terms, termSql)
	}

	return strings.Join(terms, " "+sqlOp+" ")
}
