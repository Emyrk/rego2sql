package go_rego

import (
	"fmt"
	"github.com/Emyrk/go-rego/sqlast"
	"github.com/open-policy-agent/opa/ast"
)

var _ sqlast.VariableMatcher = ACLGroupVar{}

type ACLGroupVar struct {
	StructSQL  string
	StructPath []string
	// DenyAll is helpful for when we don't care about ACL groups.
	// We need to default to denying access.
	DenyAll bool

	// FieldReference handles referencing the subfields, which could be
	// more variables. We pass one in as the global one might not be correctly
	// scoped.
	FieldReference sqlast.VariableMatcher

	// Instance fields
	Source    sqlast.RegoSource
	GroupNode sqlast.Node
}

func ACLGroupMatcher(fieldRefernce sqlast.VariableMatcher, structSQL string, structPath []string) ACLGroupVar {
	return ACLGroupVar{StructSQL: structSQL, StructPath: structPath, FieldReference: fieldRefernce}
}

func (ACLGroupVar) UseAs() sqlast.Node { return ACLGroupVar{} }
func (g *ACLGroupVar) Disable() *ACLGroupVar {
	g.DenyAll = true
	return g
}

func (g ACLGroupVar) ConvertVariable(rego ast.Ref) (sqlast.Node, bool) {
	left, err := sqlast.RegoVarPath(g.StructPath, rego)
	if err != nil {
		return nil, false
	}

	// This is what is left.
	//	{
	//	 "all_users": ["read"]
	//	}

	// We expect a group name
	if len(left) == 0 {
		return nil, false
	}

	aclGrp := ACLGroupVar{
		DenyAll:        g.DenyAll,
		StructSQL:      g.StructSQL,
		StructPath:     g.StructPath,
		FieldReference: g.FieldReference,

		Source: sqlast.RegoSource(rego.String()),
	}

	// We expect 1 more term. Either a ref or a string.
	if len(left) != 1 {
		return nil, false
	}

	// If the remaining is a variable, then we need to convert it.
	// Assuming we support variable fields.
	ref, ok := left[0].Value.(ast.Ref)
	if ok && g.FieldReference != nil {
		groupNode, ok := g.FieldReference.ConvertVariable(ref)
		if ok {
			aclGrp.GroupNode = groupNode
			return aclGrp, true
		}
	}

	// If it is a string, we assume it is a literal
	groupName, ok := left[0].Value.(ast.String)
	if ok {
		aclGrp.GroupNode = sqlast.String(string(groupName))
		return aclGrp, true
	}

	// If we have not matched it yet, then it is something we do not recognize.
	return nil, false
}

func (g ACLGroupVar) SQLString(cfg *sqlast.SQLGenerator) string {
	if g.DenyAll {
		return "false"
	}
	return fmt.Sprintf("%s->%s", g.StructSQL, g.GroupNode.SQLString(cfg))
}

func (g ACLGroupVar) ContainsSQL(cfg *sqlast.SQLGenerator, other sqlast.Node) (string, error) {
	if g.DenyAll {
		return "false", nil
	}

	switch other.UseAs().(type) {
	case sqlast.AstString:
		return fmt.Sprintf("%s ? %s", g.SQLString(cfg), other.SQLString(cfg)), nil
	}

	return "", fmt.Errorf("unsupported acl group contains %T", other)
}
