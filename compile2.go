package go_rego

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Emyrk/go-rego/sqlast"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
)

type ConvertConfig struct {
	// ConvertVariable is called each time a var is encountered. A function must be
	// returned that is a Node. The variable Node handles how to do its own
	// SQL conversion.
	ConvertVariable func(rego ast.Ref) (sqlast.Node, error)
}

func ConvertRegoAst(cfg ConvertConfig, partial *rego.PartialQueries) (sqlast.BooleanNode, error) {
	if len(partial.Queries) == 0 {
		// Always deny
		return sqlast.Bool(false), nil
	}

	for _, q := range partial.Queries {
		// An empty query in rego means "true"
		if len(q) == 0 {
			// Always allow
			return sqlast.Bool(true), nil
		}
	}

	var queries []sqlast.BooleanNode
	var builder strings.Builder
	for i, q := range partial.Queries {
		converted, err := convertQuery(cfg, q)
		if err != nil {
			return nil, fmt.Errorf("query %s: %w", q.String(), err)
		}

		boolConverted, ok := converted.(sqlast.BooleanNode)
		if !ok {
			return nil, fmt.Errorf("query %s: not a boolean", q.String())
		}

		if i != 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(q.String())
		queries = append(queries, boolConverted)
	}

	return sqlast.Or(sqlast.RegoSource(builder.String()), queries...), nil
}

func convertQuery(cfg ConvertConfig, q ast.Body) (sqlast.BooleanNode, error) {
	for _, e := range q {
		_, err := convertExpression(cfg, e)
		if err != nil {
			return nil, fmt.Errorf("expression %s: %w", e.String(), err)
		}
	}

	return nil, nil
}

func convertExpression(cfg ConvertConfig, e *ast.Expr) (sqlast.BooleanNode, error) {
	if !e.IsCall() {
		// We can only handle this if it is a single term.
		if term, ok := e.Terms.(*ast.Term); ok {
			ty, err := convertTerm(cfg, term)
			if err != nil {
				return nil, fmt.Errorf("convert term %s: %w", term.String(), err)
			}

			tyBool, ok := ty.(sqlast.BooleanNode)
			if !ok {
				return nil, fmt.Errorf("convert term %s is not a boolean: %w", term.String(), err)
			}

			return tyBool, nil
		}
		return nil, fmt.Errorf("not a call, not yet supported")
	}

	// If the expression is not a call, that means it is an operator.
	op := e.Operator().String()
	switch op {
	case "neq", "eq", "equals":
		args, err := convertTerms(cfg, e.Operands(), 2)
		if err != nil {
			return nil, fmt.Errorf("arguments: %w", err)
		}

		return sqlast.Equality(op == "neq", args[0], args[1]), nil
	//case "internal.member_2":

	default:
		return nil, fmt.Errorf("operator %s not supported", op)
	}
}

func convertTerms(cfg ConvertConfig, terms []*ast.Term, expected int) ([]sqlast.Node, error) {
	if len(terms) != expected {
		return nil, fmt.Errorf("expected %d terms, got %d", expected, len(terms))
	}

	return nil, nil
}

func convertTerm(cfg ConvertConfig, term *ast.Term) (sqlast.Node, error) {
	source := sqlast.RegoSource(term.String())
	switch t := term.Value.(type) {
	case ast.Var:
		return nil, fmt.Errorf("var not yet supported")
	case ast.Ref:
		if len(t) == 0 {
			// A reference with no text is a variable with no name?
			// This makes no sense.
			return nil, fmt.Errorf("empty ref not supported")
		}

		first, ok := t[0].Value.(ast.Var)
		if !ok {
			return nil, fmt.Errorf("ref must start with a var, got %T", t[0])
		}

		var _ = first

		// The structure of references is as follows:
		// 1. All variables start with a regoAst.Var as the first term.
		// 2. The next term is either a regoAst.String or a regoAst.Var.
		//	- regoAst.String if a static field name or index.
		//	- regoAst.Var if the field reference is a variable itself. Such as
		//    the wildcard "[_]"
		// 3. Repeat 1-2 until the end of the reference.
		node, err := cfg.ConvertVariable(t)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", t.String(), err)
		}
		return node, err
	case ast.String:
		return sqlast.String(string(t)), nil
	case ast.Number:
		return sqlast.Number(source, json.Number(t)), nil
	case ast.Boolean:
		return sqlast.Bool(bool(t)), nil
	case *ast.Array:
		elems := make([]sqlast.Node, 0, t.Len())
		for i := 0; i < t.Len(); i++ {
			value, err := convertTerm(cfg, t.Elem(i))
			if err != nil {
				return nil, fmt.Errorf("array element %d in %q: %w", i, t.String(), err)
			}
			elems = append(elems, value)
		}
		return sqlast.Array(source, elems...)
	case ast.Object:
		return nil, fmt.Errorf("object not yet supported")
	case ast.Set:
		// Just treat a set like an array for now.
		arr := t.Sorted()
		return convertTerm(cfg, &ast.Term{
			Value:    arr,
			Location: term.Location,
		})
	default:
		return nil, fmt.Errorf("%T not yet supported", t)
	}
}
