package sqlast

import (
	"fmt"
	"strconv"
)

type AstBoolean struct {
	Source RegoSource
	Value  bool
}

func Bool(t bool) BooleanNode {
	return AstBoolean{Value: t, Source: RegoSource(strconv.FormatBool(t))}
}

func (AstBoolean) IsBooleanNode() {}
func (AstBoolean) UseAs() Node    { return AstBoolean{} }

func (b AstBoolean) SQLString(cfg *SQLGenerator) string {
	return strconv.FormatBool(b.Value)
}

func (b AstBoolean) EqualsSQLString(cfg *SQLGenerator, not bool, other Node) (string, error) {
	switch other.UseAs().(type) {
	case AstBoolean:
		return basicSQLEquality(cfg, not, b, other), nil
	case BooleanNode:
		return fmt.Sprintf("%s %s (%s)",
			b.SQLString(cfg),
			equalsOp(not),
			other.SQLString(cfg),
		), nil
	}

	return "", fmt.Errorf("unsupported equality: %T %s %T", b, equalsOp(not), other)

}
