package go_rego

import (
	"fmt"
	"strconv"
)

type Equaler interface {
	Equal(not bool, other SQLType) (string, error)
}

func mathEquals(not bool, a, b string) string {
	sign := "="
	if not {
		sign = "!="
	}
	return fmt.Sprintf("%s %s %s", a, sign, b)
}

func (a String) Equal(not bool, other SQLType) (string, error) {
	switch b := other.(type) {
	case Ref:
		if typeEquals(a, b.VarType) {
			return mathEquals(not, sqlQuote(a.Value), b.NameFunc(b.Path)), nil
		}

		if len(b.PathLeft) == 1 && b.PathLeft[0] == wildcard && typeEquals(b.VarType, Array{}) {
			return mathEquals(not, sqlQuote(a.Value), fmt.Sprintf("ANY(%s)", b.NameFunc(b.PathLeft))), nil
		}
		return "", fmt.Errorf("cannot compare ref %T to %T", a, b.VarType)
	case String:
		return mathEquals(not, sqlQuote(a.Value), sqlQuote(b.Value)), nil
	default:
		return "", fmt.Errorf("cannot compare %T to %T", a, b)
	}
}

func (a Boolean) Equal(not bool, other SQLType) (string, error) {
	switch b := other.(type) {
	case Ref:
		if typeEquals(b.VarType, a) {
			return mathEquals(not, strconv.FormatBool(a.Value), b.NameFunc(b.Path)), nil
		}
		return "", fmt.Errorf("cannot compare ref %T to %T", a, b.VarType)
	case Boolean:
		return mathEquals(not, strconv.FormatBool(a.Value), strconv.FormatBool(b.Value)), nil
	default:
		return "", fmt.Errorf("cannot compare %T to %T", a, b)
	}
}

func sqlQuote(s string) string {
	return fmt.Sprintf("'%s'", s)
}
