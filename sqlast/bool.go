package sqlast

import (
	"fmt"
	"strconv"
)

type boolean struct {
	Source RegoSource
	Value  bool
}

func Bool(t bool) BooleanNode {
	return boolean{Value: t, Source: RegoSource(strconv.FormatBool(t))}
}

func (b boolean) SQLString(cfg *SQLGenerator) string {
	return strconv.FormatBool(b.Value)
}

func (b boolean) EqualsSQLString(cfg *SQLGenerator, not bool, other Node) string {
	switch other.(type) {
	case boolean:
		return basicSQLEquality(cfg, not, b, other)
	case BooleanNode:
		return fmt.Sprintf("%s %s (%s)",
			b.SQLString(cfg),
			equalsOp(not),
			other.SQLString(cfg),
		)
	}

	cfg.AddError(fmt.Errorf("unsupported equality: %T %s %T", b, equalsOp(not), other))
	return "EqualityError"
}
