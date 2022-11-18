package sqlast

import (
	"fmt"
)

// SupportsContains is an interface that can be implemented by types that
// support "me.Contains(other)". This is `internal_member2` in the rego.
type SupportsContains interface {
	ContainsSQL(cfg *SQLGenerator, other Node) string
}

var _ BooleanNode = memberOf{}
var _ Node = memberOf{}

//var _ SupportsMemberOf = memberOf{}

type memberOf struct {
	Left  Node
	Right Node

	// Not just inverses the result of the comparison. We could implement this
	// as a Not node wrapping the equality, but this is more efficient.
	Not bool
}

func MemberOf(notEquals bool, a, b Node) BooleanNode {
	return memberOf{
		Left:  a,
		Right: b,
		Not:   notEquals,
	}
}

func (e memberOf) SQLString(cfg *SQLGenerator) string {
	// Equalities can be flipped without changing the result, so we can
	// try both left = right and right = left.
	if eq, ok := e.Left.(SupportsEquality); ok {
		return eq.EqualsSQLString(cfg, e.Not, e.Right)
	}

	if eq, ok := e.Right.(SupportsEquality); ok {
		return eq.EqualsSQLString(cfg, e.Not, e.Left)
	}

	cfg.AddError(fmt.Errorf("unsupported equality: %T %s %T", e.Left, equalsOp(e.Not), e.Right))
	return "EqualityError"
}
