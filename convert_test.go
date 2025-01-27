package rego2sql_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Emyrk/rego2sql"
	"github.com/Emyrk/rego2sql/codercfg"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestEx(t *testing.T) {
	//tree, err := pg_query.Parse("SELECT * WHERE 1 = ANY(ARRAY[1,2,3])")
	tree, err := pg_query.Parse("SELECT * WHERE group_acl->'organization_id' ? 'read'")
	require.NoError(t, err)

	where := tree.GetStmts()[0].GetStmt()
	fmt.Println(where.String())
}

// TestRegoQueriesNoVariables handles cases without variables. These should be
// very simple and straight forward.
func TestRegoQueries(t *testing.T) {

	defConverts := func() *rego2sql.VariableConverter {
		matcher := rego2sql.NewVariableConverter().RegisterMatcher(
			// Basic strings
			// "organization_id :: text"
			rego2sql.StringVarMatcher([]string{"input", "object", "org_owner"}, []string{"organization_id"}, cty.UnknownVal(cty.String)),
			rego2sql.StringVarMatcher([]string{"input", "object", "owner"}, []string{"owner"}, cty.UnknownVal(cty.String)),
		)
		matcher.RegisterMatcher(
			codercfg.GroupACLMatcher(matcher),
			codercfg.UserACLMatcher(matcher),
		)

		return matcher
	}

	noACLs := func() *rego2sql.VariableConverter {
		matcher := rego2sql.NewVariableConverter().RegisterMatcher(
			// Basic strings
			// "organization_id :: text"
			rego2sql.StringVarMatcher([]string{"input", "object", "org_owner"}, []string{"organization_id"}, cty.UnknownVal(cty.String)),
			rego2sql.StringVarMatcher([]string{"input", "object", "owner"}, []string{"owner"}, cty.UnknownVal(cty.String)),
		)

		return matcher
	}

	testCases := []struct {
		Name                 string
		Queries              []string
		ExpectedSQL          string
		ExpectError          bool
		ExpectedSQLGenErrors int

		VariableConverter rego2sql.VariableMatcher
		UnknownVarsFalse  bool
	}{
		{
			Name:        "Empty",
			Queries:     []string{``},
			ExpectedSQL: "true",
		},
		{
			Name:        "True",
			Queries:     []string{`true`},
			ExpectedSQL: "(true)",
		},
		{
			Name:        "False",
			Queries:     []string{`false`},
			ExpectedSQL: "(false)",
		},
		{
			Name:        "BoolOps",
			Queries:     []string{"5 = 5"},
			ExpectedSQL: "(5 = 5)",
		},
		{
			Name:        "MultipleBool",
			Queries:     []string{"true", "false"},
			ExpectedSQL: "(true) OR (false)",
		},
		{
			Name: "Numbers",
			Queries: []string{
				"(1 != 2) = true",
				"5 == 5",
			},
			ExpectedSQL: "((1 <> 2) = true) OR (5 = 5)",
		},
		// Variables
		{
			// Always return a constant string for all variables.
			Name: "V_Basic",
			Queries: []string{
				`input.x = "hello_world"`,
			},
			ExpectedSQL: "(col_ref = 'hello_world')",
			VariableConverter: rego2sql.NewVariableConverter().RegisterMatcher(
				rego2sql.StringVarMatcher([]string{"input", "x"}, []string{
					"col_ref",
				}, cty.UnknownVal(cty.String)),
			),
		},
		// Coder Variables
		{
			// Always return a constant string for all variables.
			Name: "GroupACLConstant",
			Queries: []string{
				`"read" in input.object.acl_group_list.allUsers`,
			},
			ExpectedSQL:       "((group_acl -> 'allUsers') ? 'read')",
			VariableConverter: defConverts(),
		},
		{
			// Always return a constant string for all variables.
			Name: "GroupACL",
			Queries: []string{
				`"read" in input.object.acl_group_list[input.object.org_owner]`,
			},
			ExpectedSQL:       "((group_acl -> organization_id) ? 'read')",
			VariableConverter: defConverts(),
		},
		{
			Name: "VarInArray",
			Queries: []string{
				`input.object.org_owner in {"a", "b", "c"}`,
			},
			ExpectedSQL:       "(organization_id = ANY(ARRAY['a', 'b', 'c']))",
			VariableConverter: defConverts(),
		},
		{
			Name: "MultipleExpr",
			Queries: []string{
				`
					input.object.org_owner in {"a", "b", "c"}
					input.object.owner != ""
				`,
			},
			ExpectedSQL:       "(organization_id = ANY(ARRAY['a', 'b', 'c']) AND owner <> '')",
			VariableConverter: defConverts(),
		},
		{
			Name: "Complex",
			Queries: []string{
				`input.object.org_owner != ""`,
				`input.object.org_owner in {"a", "b", "c"}`,
				`input.object.org_owner != ""`,
				`"read" in input.object.acl_group_list.allUsers`,
				`"read" in input.object.acl_user_list.me`,
			},
			ExpectedSQL: `(organization_id <> '') OR ` +
				`(organization_id = ANY(ARRAY['a', 'b', 'c'])) OR ` +
				`(organization_id <> '') OR ` +
				`((group_acl -> 'allUsers') ? 'read') OR ` +
				`((user_acl -> 'me') ? 'read')`,
			VariableConverter: defConverts(),
		},
		{
			Name: "NoACLs",
			Queries: []string{
				`"read" in input.object.acl_group_list[input.object.org_owner]`,
			},
			ExpectedSQL:       "false",
			VariableConverter: noACLs(),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			part := partialQueries(t, tc.Queries...)

			cfg := rego2sql.ConvertConfig{
				VariableConverter: tc.VariableConverter,
				UnknownVarsFalse:  tc.UnknownVarsFalse,
			}

			requireConvert(t, convertTestCase{
				part:               part,
				cfg:                cfg,
				expectSQL:          tc.ExpectedSQL,
				expectConvertError: tc.ExpectError,
				expectSQLGenErrors: tc.ExpectedSQLGenErrors,
			})
		})
	}
}

type convertTestCase struct {
	part *rego.PartialQueries
	cfg  rego2sql.ConvertConfig

	expectConvertError bool
	expectSQL          string
	expectSQLGenErrors int
}

func requireConvert(t *testing.T, tc convertTestCase) {
	t.Helper()

	for i, q := range tc.part.Queries {
		t.Logf("Query %d: %s", i, q.String())
	}
	for i, s := range tc.part.Support {
		t.Logf("Support %d: %s", i, s.String())
	}

	sqlNode, err := rego2sql.Convert(tc.cfg, tc.part.Queries)
	if tc.expectConvertError {
		require.Error(t, err)
	} else {
		require.NoError(t, err, "compile")

		gen, err := rego2sql.Serialize(sqlNode)
		require.NoError(t, err, "serialize")

		require.Equal(t, tc.expectSQL, gen, "sql match")
	}
}

func partialQueries(t *testing.T, queries ...string) *rego.PartialQueries {
	opts := ast.ParserOptions{
		AllFutureKeywords: true,
	}

	astQueries := make([]ast.Body, 0, len(queries))
	for _, q := range queries {
		astQueries = append(astQueries, ast.MustParseBodyWithOpts(q, opts))
	}

	prepareQueries := make([]rego.PreparedEvalQuery, 0, len(queries))
	for _, q := range astQueries {
		var prepped rego.PreparedEvalQuery
		var err error
		if q.String() == "" {
			prepped, err = rego.New(
				rego.Query("true"),
			).PrepareForEval(context.Background())
		} else {
			prepped, err = rego.New(
				rego.ParsedQuery(q),
			).PrepareForEval(context.Background())
		}
		require.NoError(t, err, "prepare query")
		prepareQueries = append(prepareQueries, prepped)
	}
	return &rego.PartialQueries{
		Queries: astQueries,
		Support: []*ast.Module{},
	}
}
