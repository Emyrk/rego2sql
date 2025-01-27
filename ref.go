package rego2sql

import (
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/zclconf/go-cty/cty"
)

type VariableMatcher interface {
	ConvertVariable(rego ast.Ref) (*Item, bool)
}

type VariableConverter struct {
	converters []VariableMatcher
}

func NewVariableConverter() *VariableConverter {
	return &VariableConverter{}
}

func (vc *VariableConverter) RegisterMatcher(m ...VariableMatcher) *VariableConverter {
	vc.converters = append(vc.converters, m...)
	// Returns the VariableConverter for easier instantiation
	return vc
}

func (vc *VariableConverter) ConvertVariable(rego ast.Ref) (*Item, bool) {
	for _, c := range vc.converters {
		if n, ok := c.ConvertVariable(rego); ok {
			return n, true
		}
	}
	return nil, false
}

// RegoVarPath will consume the following terms from the given rego Ref and
// return the remaining terms. If the path does not fully match, an error is
// returned. The first term must always be a Var.
func RegoVarPath(path []string, terms []*ast.Term) ([]*ast.Term, error) {
	if len(terms) < len(path) {
		return nil, fmt.Errorf("path %s longer than rego path %s", path, terms)
	}

	varTerm, ok := terms[0].Value.(ast.Var)
	if !ok {
		return nil, fmt.Errorf("expected var, got %T", terms[0])
	}

	if string(varTerm) != path[0] {
		return nil, fmt.Errorf("expected var %s, got %s", path[0], varTerm)
	}

	for i := 1; i < len(path); i++ {
		nextTerm, ok := terms[i].Value.(ast.String)
		if !ok {
			return nil, fmt.Errorf("expected ast.string, got %T", terms[i])
		}

		if string(nextTerm) != path[i] {
			return nil, fmt.Errorf("expected string %s, got %s", path[i], nextTerm)
		}
	}

	return terms[len(path):], nil
}

// astStringVar is any variable that represents a string.
type astStringVar struct {
	FieldPath    []string
	ColumnString []string
	Typ          cty.Type
}

func StringVarMatcher(regoPath []string, columnRef []string, typ cty.Type) VariableMatcher {
	return astStringVar{
		FieldPath:    regoPath,
		ColumnString: columnRef,
		Typ:          typ,
	}
}

// ConvertVariable will return a new astStringVar Node if the given rego Ref
// matches this astStringVar.
func (s astStringVar) ConvertVariable(rego ast.Ref) (*Item, bool) {
	left, err := RegoVarPath(s.FieldPath, rego)
	if err == nil && len(left) == 0 {
		fields := make([]*pg_query.Node, 0, len(s.ColumnString))
		for _, p := range s.ColumnString {
			fields = append(fields, pg_query.MakeStrNode(p))
		}

		return &Item{
			Node: pg_query.MakeColumnRefNode(fields, 0),
			Type: s.Typ,
		}, true
	}

	return nil, false
}
