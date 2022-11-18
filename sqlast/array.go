package sqlast

import (
	"fmt"
	"reflect"
	"strings"
)

type array struct {
	Source RegoSource
	Value  []Node
}

func Array(source RegoSource, nodes ...Node) (Node, error) {
	for i := 1; i < len(nodes); i++ {
		if reflect.TypeOf(nodes[0]) != reflect.TypeOf(nodes[i]) {
			// Do not allow mixed types in arrays
			return nil, fmt.Errorf("array element %d in %q: type mismatch", i, source)
		}
	}
	return array{Value: nodes, Source: source}, nil
}

func (a array) ContainsSQL(cfg *SQLGenerator, other Node) string {
	// TODO: Handle array.Contains(array). Must handle types correctly.
	// Should implement as strict subset.

	if reflect.TypeOf(a.MyType()) != reflect.TypeOf(other) {
		cfg.AddError(fmt.Errorf("array contains %q: type mismatch (%T, %T)",
			a.Source, a.MyType(), other))
		return "ArrayContainsError"
	}

	return fmt.Sprintf("%s = ANY(%s)", other.SQLString(cfg), a.SQLString(cfg))
}

func (a array) SQLString(cfg *SQLGenerator) string {
	switch a.MyType().(type) {
	case invalidNode:
		cfg.AddError(fmt.Errorf("array %q: empty array", a.Source))
		return "ArrayError"
	case number, astString, boolean:
		// Primitive types
		values := make([]string, 0, len(a.Value))
		for _, v := range a.Value {
			values = append(values, v.SQLString(cfg))
		}
		return fmt.Sprintf("ARRAY[%s]", strings.Join(values, ", "))
	}

	cfg.AddError(fmt.Errorf("array %q: unsupported type %T", a.Source, a.MyType()))
	return "ArrayError"
}

func (a array) MyType() Node {
	if len(a.Value) == 0 {
		return invalidNode{}
	}
	return a.Value[0]
}
