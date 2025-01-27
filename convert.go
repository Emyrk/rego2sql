package rego2sql

import (
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

type ConvertConfig struct {
	// VariableConverter is called each time a var is encountered. This creates
	// the SQL ast for the variable.
	VariableConverter VariableMatcher
}

func Convert(cfg ConvertConfig, queries []ast.Body) (*pg_query.Node, error) {
	// the rego policy is false if no queries exist to satisfy
	if len(queries) == 0 {
		return constBoolean(false, 0), nil
	}

	// All partial queries are OR'd together
	// If any of them have a length of 0, then that query is 'true'.
	// Which means the policy is 'true'.
	for _, q := range queries {
		if len(q) == 0 {
			return constBoolean(true, 0), nil
		}
	}

	crv := &converter{
		stack: newStack[*Item](),
	}

	// A list of all the nodes that will be OR'd together
	nodes := make([]*pg_query.Node, 0, len(queries))
	for _, q := range queries {
		qn, err := crv.convertQuery(cfg, q)
		if err != nil {
			return nil, fmt.Errorf("convert query: %w", err)
		}
		nodes = append(nodes, qn)
	}

	orJoined := pg_query.MakeBoolExprNode(pg_query.BoolExprType_OR_EXPR, nodes, 0)
	return orJoined, nil
}

func constBoolean(val bool, location int32) *pg_query.Node {
	return &pg_query.Node{
		Node: &pg_query.Node_AConst{
			AConst: &pg_query.A_Const{
				Val: &pg_query.A_Const_Boolval{
					Boolval: &pg_query.Boolean{
						Boolval: val,
					},
				},
				Isnull:   false,
				Location: location,
			},
		},
	}
}

func constFloat(float string, location int32) *pg_query.Node {
	return &pg_query.Node{
		Node: &pg_query.Node_AConst{
			AConst: &pg_query.A_Const{
				Val: &pg_query.A_Const_Fval{
					Fval: &pg_query.Float{
						Fval: float,
					},
				},
				Isnull:   false,
				Location: location,
			},
		},
	}
}
