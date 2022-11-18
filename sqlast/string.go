package sqlast

import "fmt"

type astString struct {
	Source RegoSource
	Value  string
}

func String(v string) Node {
	return astString{Value: v, Source: RegoSource(v)}
}

func (s astString) SQLString(cfg *SQLGenerator) string {
	return "'" + s.Value + "'"
}

func (s astString) EqualsSQLString(cfg *SQLGenerator, not bool, other Node) string {
	switch other.(type) {
	case astString:
		return basicSQLEquality(cfg, not, s, other)
	}

	cfg.AddError(fmt.Errorf("unsupported equality: %T %s %T", s, equalsOp(not), other))
	return "EqualityError"
}
