package codercfg

import (
	"github.com/Emyrk/rego2sql"
	"github.com/open-policy-agent/opa/v1/ast"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/zclconf/go-cty/cty"
)

// ACLMatcher is a variable matcher that handles group_acl and user_acl.
// The sql type is a jsonb object with the following structure:
//
//	"group_acl": {
//	 "<group_name>": ["<actions>"]
//	}
//
// This is a custom variable matcher as json objects have arbitrary complexity.
type ACLMatcher struct {
	StructSQL string
	// input.object.group_acl -> ["input", "object", "group_acl"]
	RegoPath  []string
	ColumnRef []string

	// FieldReference handles referencing the subfields, which could be
	// more variables. We pass one in as the global one might not be correctly
	// scoped.
	FieldReference rego2sql.VariableMatcher
}

func ACLGroupMatcher(fieldReference rego2sql.VariableMatcher, regoPath []string, columnRef []string) ACLMatcher {
	return ACLMatcher{RegoPath: regoPath, ColumnRef: columnRef, FieldReference: fieldReference}
}

func (g ACLMatcher) ConvertVariable(rego ast.Ref) (*rego2sql.Item, bool) {
	// "left" will be a map of group names to actions in rego.
	//	{
	//	 "all_users": ["read"]
	//	}
	left, err := rego2sql.RegoVarPath(g.RegoPath, rego)
	if err != nil {
		return nil, false
	}

	// We expect 1 more term. Either a ref or a string.
	if len(left) != 1 {
		return nil, false
	}

	fields := make([]*pg_query.Node, 0, len(g.ColumnRef))
	for _, p := range g.ColumnRef {
		fields = append(fields, pg_query.MakeStrNode(p))
	}

	l := pg_query.MakeColumnRefNode(fields, 0)
	var r *rego2sql.Item

	// If the remaining is a variable, then we need to convert it.
	// Assuming we support variable fields.
	ref, ok := left[0].Value.(ast.Ref)
	if ok && g.FieldReference != nil {
		lastRefNode, ok := g.FieldReference.ConvertVariable(ref)
		if !ok {
			return nil, false
		}
		r = lastRefNode
	}

	if r == nil {
		// If it is a string, we assume it is a literal
		groupName, ok := left[0].Value.(ast.String)
		if ok {
			r = &rego2sql.Item{
				Node:  pg_query.MakeAConstStrNode(string(groupName), 0),
				Value: cty.StringVal(string(groupName)),
			}
		}
	}

	return &rego2sql.Item{
		Node:   pg_query.MakeAExprNode(pg_query.A_Expr_Kind_AEXPR_OP, []*pg_query.Node{pg_query.MakeStrNode("->")}, l, r.Node, 0),
		Value:  rego2sql.MarkJSONB(cty.ListValEmpty(cty.String)), // really a uuid, and really json...
		Source: rego.String(),
	}, true
}
