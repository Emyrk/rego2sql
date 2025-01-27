package rego2sql

import (
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/zclconf/go-cty/cty"
)

const (
	stopVisiting     = true
	continueVisiting = false
)

type converter struct {
	stack *stack[*Item]
}

func (c *converter) convertQuery(cfg ConvertConfig, q ast.Body) (*pg_query.Node, error) {
	var visitErr error

	ast.NewGenericVisitor(func(n interface{}) bool {
		switch val := n.(type) {
		case *ast.Expr:
			if val.IsCall() {
				node, err := convertCall(cfg, val.Terms.([]*ast.Term))
				if err != nil {
					visitErr = fmt.Errorf("convert call %s: %w", val.String(), err)
					return stopVisiting
				}
				c.stack.Push(node)
				return stopVisiting
			}

			// TODO: Check other types of expressions
			return continueVisiting
		case ast.Body:
		case ast.Call:
			return stopVisiting
		case *ast.Term:
			node, err := convertTerm(cfg, val)
			if err != nil {
				visitErr = fmt.Errorf("convert term %s: %w", val.String(), err)
				return stopVisiting
			}
			c.stack.Push(node)
			return stopVisiting
		default:
			visitErr = fmt.Errorf("unsupported type %T", n)
			return stopVisiting
		}
		return continueVisiting
	}).Walk(q)
	if visitErr != nil {
		return nil, visitErr
	}

	// Join all nodes with AND
	if c.stack.Len() == 0 {
		return nil, fmt.Errorf("stack is empty, no sql query generated")
	}

	nodes := make([]*pg_query.Node, 0, c.stack.Len())
	for !c.stack.IsEmpty() {
		sn := c.stack.Pop()
		if sn.Value.Type() != cty.Bool {
			return nil, fmt.Errorf("expected boolean type, got %s for rego %q", sn.Value, sn.Source)
		}
		nodes = append(nodes, sn.Node)
	}

	return pg_query.MakeBoolExprNode(pg_query.BoolExprType_AND_EXPR, nodes, 0), nil
}

// convertCall converts a function call to a SQL expression.
func convertCall(cfg ConvertConfig, call ast.Call) (*Item, error) {
	if len(call) == 0 {
		return nil, fmt.Errorf("empty call")
	}

	// Operator is the first term
	op := call[0]
	var args []*ast.Term
	if len(call) > 1 {
		args = call[1:]
	}

	opString := op.String()
	// Supported operators.
	switch op.String() {
	case "neq", "eq", "equals", "equal":
		termArgs, err := convertTerms(cfg, args, 2)
		if err != nil {
			return nil, fmt.Errorf("arguments: %w", err)
		}

		if !termArgs[0].Value.Type().Equals(termArgs[1].Value.Type()) {
			return nil, fmt.Errorf("arguments are not the same type for equality: %q",
				call.String())
		}

		sqlOp := "="
		if opString == "neq" || opString == "notequals" || opString == "notequal" {
			sqlOp = "<>"
		}

		return &Item{
			Node: pg_query.MakeAExprNode(pg_query.A_Expr_Kind_AEXPR_OP,
				[]*pg_query.Node{pg_query.MakeStrNode(sqlOp)},
				termArgs[0].Node, termArgs[1].Node, 0,
			),
			Value:  cty.UnknownVal(cty.Bool),
			Source: call.String(),
		}, nil
	case "internal.member_2":
		termArgs, err := convertTerms(cfg, args, 2)
		if err != nil {
			return nil, fmt.Errorf("arguments: %w", err)
		}

		if termArgs[1].Value.Type().IsListType() {
			// TODO: Probably handle more json types better. This is hard coded
			// for how we do it in coder.
			if IsJSONBool(termArgs[1].Value) {
				// Use '?' operator for JSONB
				return &Item{
					Node: pg_query.MakeAExprNode(pg_query.A_Expr_Kind_AEXPR_OP,
						[]*pg_query.Node{pg_query.MakeStrNode("?")},
						termArgs[1].Node, termArgs[0].Node, 0,
					),
					Value:  cty.UnknownVal(cty.Bool),
					Source: call.String(),
				}, nil
			}

			return &Item{
				Node: pg_query.MakeAExprNode(pg_query.A_Expr_Kind_AEXPR_OP_ANY,
					[]*pg_query.Node{pg_query.MakeStrNode("=")},
					termArgs[0].Node, termArgs[1].Node, 0,
				),
				Value:  cty.UnknownVal(cty.Bool),
				Source: call.String(),
			}, nil
		}

		return nil, fmt.Errorf("member_2: second argument is not a list: %q", call.String())
	default:
		return nil, fmt.Errorf("operator %s not supported", op)
	}
}

func convertTerm(cfg ConvertConfig, term *ast.Term) (*Item, error) {
	source := term.String()
	switch val := term.Value.(type) {
	case ast.Var:
		return nil, fmt.Errorf("var not yet supported")
	case ast.Ref:
		if len(val) == 0 {
			// A reference with no text is a variable with no name?
			// This makes no sense.
			return nil, fmt.Errorf("empty ref not supported")
		}

		if cfg.VariableConverter == nil {
			return nil, fmt.Errorf("variable converter not set, ref %q cannot be handled", val.String())
		}

		// The structure of references is as follows:
		// 1. All variables start with a regoAst.Var as the first term.
		// 2. The next term is either a regoAst.String or a regoAst.Var.
		//	- regoAst.String if a static field name or index.
		//	- regoAst.Var if the field reference is a variable itself. Such as
		//    the wildcard "[_]"
		// 3. Repeat 1-2 until the end of the reference.
		node, ok := cfg.VariableConverter.ConvertVariable(val)
		if !ok {
			return nil, fmt.Errorf("variable %q cannot be converted", val.String())
		}
		return node, nil
	case ast.String:
		return &Item{
			Node:   pg_query.MakeAConstStrNode(string(val), 0),
			Value:  cty.StringVal(string(val)),
			Source: val.String(),
		}, nil
	case ast.Number:
		i, iOk := val.Int64()
		f, fOk := val.Float64()
		if !iOk && !fOk {
			return nil, fmt.Errorf("convert to integer: %q", source)
		}
		if iOk {
			return &Item{
				Node:   pg_query.MakeAConstIntNode(i, 0),
				Value:  cty.NumberIntVal(i),
				Source: val.String(),
			}, nil
		}

		return &Item{
			Node:   constFloat(val.String(), 0),
			Value:  cty.NumberFloatVal(f),
			Source: val.String(),
		}, nil
	case ast.Boolean:
		return &Item{
			Node:   constBoolean(bool(val), 0),
			Value:  cty.BoolVal(bool(val)),
			Source: val.String(),
		}, nil
	case *ast.Array:
		arrayType := cty.NilType

		ctyList := make([]cty.Value, 0, val.Len())
		elemNodes := make([]*pg_query.Node, 0, val.Len())
		elems := make([]*Item, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			value, err := convertTerm(cfg, val.Elem(i))
			if err != nil {
				return nil, fmt.Errorf("array element %d in %q: %w", i, val.String(), err)
			}
			if i == 0 {
				arrayType = value.Value.Type()
			} else {
				if !value.Value.Type().Equals(arrayType) {
					return nil, fmt.Errorf("array of mixed types, this is unsupported: %q", val.String())
				}
			}
			elems = append(elems, value)
			elemNodes = append(elemNodes, value.Node)
			ctyList = append(ctyList, value.Value)
		}

		return &Item{
			Node: &pg_query.Node{
				Node: &pg_query.Node_AArrayExpr{
					AArrayExpr: &pg_query.A_ArrayExpr{
						Elements: elemNodes,
						Location: 0,
					},
				},
			},
			Value:  cty.ListVal(ctyList),
			Source: val.String(),
		}, nil
	case ast.Object:
		return nil, fmt.Errorf("object not yet supported")
	case ast.Set:
		// Just treat a set like an array for now.
		arr := val.Sorted()
		return convertTerm(cfg, &ast.Term{
			Value:    arr,
			Location: term.Location,
		})
	case ast.Call:
		return convertCall(cfg, val)
	default:
		return nil, fmt.Errorf("%T not yet supported", val)
	}
}

func convertTerms(cfg ConvertConfig, terms []*ast.Term, expected int) ([]*Item, error) {
	if len(terms) != expected {
		return nil, fmt.Errorf("expected %d terms, got %d", expected, len(terms))
	}

	result := make([]*Item, 0, len(terms))
	for _, t := range terms {
		term, err := convertTerm(cfg, t)
		if err != nil {
			return nil, fmt.Errorf("term: %w", err)
		}
		result = append(result, term)
	}

	return result, nil
}
