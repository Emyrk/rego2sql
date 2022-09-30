package go_rego

import (
	"fmt"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"reflect"
	"strconv"
	"strings"
)

type CompileConfig struct {
	VariableTypes *Tree
}

func CompileSQL(cfg CompileConfig, partial *rego.PartialQueries) (string, error) {
	if len(partial.Queries) == 0 {
		// Always deny
		return "false", nil
	}
	for _, q := range partial.Queries {
		if len(q) == 0 {
			// Always allow
			return "true", nil
		}
	}

	var builder strings.Builder
	for i, q := range partial.Queries {
		sql, err := processQuery(cfg, q)
		if err != nil {
			return "", fmt.Errorf("query %s: %w", q.String(), err)
		}
		if i != 0 {
			builder.WriteString(" OR ")
		}
		builder.WriteString(sql)
	}

	return builder.String(), nil
}

func processQuery(cfg CompileConfig, q ast.Body) (string, error) {
	var builder strings.Builder
	for i, e := range q {
		sql, err := processExpression(cfg, e)
		if err != nil {
			return "", fmt.Errorf("expression %s: %w", e.String(), err)
		}
		if i != 0 {
			builder.WriteString(" AND ")
		}
		builder.WriteString(sql)
	}
	return builder.String(), nil
}

func processExpression(cfg CompileConfig, e *ast.Expr) (string, error) {
	if !e.IsCall() {
		// A single term
		if term, ok := e.Terms.(*ast.Term); ok {
			ty, err := processTerm(cfg, term)
			if err == nil {
				switch v := ty.(type) {
				case Ref:
					if typeEquals(v.VarType, Boolean{}) {
						return v.NameFunc(v.Path), nil
					}
				case Boolean:
					return strconv.FormatBool(v.Value), nil
				}
			}
		}
		return "", fmt.Errorf("not a call, not yet supported")
	}

	op := e.Operator().String()
	switch op {
	case "neq", "eq", "equals":
		args, err := processTerms(cfg, e.Operands(), 2)
		if err != nil {
			return "", fmt.Errorf("arguments: %w", err)
		}
		first, ok := args[0].(Equaler)
		if !ok {
			return "", fmt.Errorf("'Equals()' function not implemented for type %T", args[0])
		}
		return first.Equal(op == "neq", args[1])
	}

	return "", nil
}

func processTerms(cfg CompileConfig, terms []*ast.Term, expected int) ([]SQLType, error) {
	if len(terms) != expected {
		return nil, fmt.Errorf("expected %d terms, got %d", expected, len(terms))
	}

	result := make([]SQLType, 0, len(terms))
	for _, t := range terms {
		term, err := processTerm(cfg, t)
		if err != nil {
			return nil, fmt.Errorf("term: %w", err)
		}
		result = append(result, term)
	}

	return result, nil
}

func processTerm(cfg CompileConfig, t *ast.Term) (SQLType, error) {
	regoBase := RegoBase{RegoString: t.String()}
	switch t := t.Value.(type) {
	case ast.Var:
		return nil, fmt.Errorf("var not yet supported")
	case ast.Ref:
		if len(t) == 0 {
			return nil, fmt.Errorf("empty ref not supported")
		}
		first, ok := t[0].Value.(ast.Var)
		if !ok {
			return nil, fmt.Errorf("ref must start with a var, got %T", t[0])
		}
		ref := Ref{
			base: regoBase,
			Path: []string{string(first)},
		}

		for _, elem := range t[1:] {
			switch e := elem.Value.(type) {
			case ast.String:
				ref.Path = append(ref.Path, string(e))
			default:
				return nil, fmt.Errorf("ref element type %T not supported", e)
			}
		}
		node := cfg.VariableTypes.PathNode(ref.Path)
		if node == nil {
			return nil, fmt.Errorf("unknown variable type for %s", ref.Path)
		}
		ref.VarType = node.NodeSQLType
		ref.NameFunc = node.ColumnName

		return ref, nil
	case ast.String:
		return String{
			Value: string(t),
			base:  regoBase,
		}, nil
	case ast.Number:
		return nil, fmt.Errorf("not yet supported")
	case ast.Boolean:
		return Boolean{
			Value: bool(t),
			base:  regoBase,
		}, nil
	case *ast.Array:
		arr := Array{
			base:   regoBase,
			Values: make([]SQLType, 0, t.Len()),
		}
		for i := 0; i < t.Len(); i++ {
			value, err := processTerm(cfg, t.Elem(i))
			if err != nil {
				return nil, fmt.Errorf("array element %d in %q: %w", i, t.String(), err)
			}
			if i == 0 {
				arr.elemType = value
			} else {
				if reflect.TypeOf(arr.elemType).String() != reflect.TypeOf(value).String() {
					return nil, fmt.Errorf("array element %d in %q: type mismatch", i, t.String())
				}
			}
			arr.Values = append(arr.Values, value)
		}
		return arr, nil
	case ast.Object:
		return nil, fmt.Errorf("not yet supported")
	case ast.Set:
		return nil, fmt.Errorf("not yet supported")
	default:
		return nil, fmt.Errorf("%T not yet supported", t)
	}
}
