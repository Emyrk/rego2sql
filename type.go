package go_rego

import "reflect"

type RegoBase struct {
	RegoString string
}

type SQLType interface {
}

type String struct {
	Value string
	base  RegoBase
}

//func (String) Type() string { return "string" }

type Boolean struct {
	Value bool
	base  RegoBase
}

//func (Boolean) Type() string { return "bool" }

type Array struct {
	elemType SQLType
	Values   []SQLType
	base     RegoBase
}

//func (a Array) Type() string { return fmt.Sprintf("[]%s", a.elemType.Type()) }

// Ref is a reference to a value. This is a variable
type Ref struct {
	Path []string
	// VarType is the type of the value being referenced
	VarType  SQLType
	NameFunc ColumnNameFunc

	base RegoBase
}

type Map struct {
	ValueType SQLType
	// Only support key strings for now
	Values map[string]SQLType
	base   RegoBase
}

func typeEquals(a, b SQLType) bool {
	return reflect.TypeOf(a).String() == reflect.TypeOf(b).String()
}
