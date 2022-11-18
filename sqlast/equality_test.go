package sqlast_test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEquality(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name           string
		Equality       sqlast.Node
		ExpectedSQL    string
		ExpectedErrors int
	}{
		{
			Name: "String=String",
			Equality: sqlast.Equality(false,
				sqlast.String("foo"),
				sqlast.String("bar"),
			),
			ExpectedSQL: "'foo' = 'bar'",
		},
		{
			Name: "String=Equality",
			Equality: sqlast.Equality(false,
				sqlast.Bool(true),
				sqlast.Equality(false,
					sqlast.String("foo"),
					sqlast.String("foo"),
				),
			),
			ExpectedSQL: "true = ('foo' = 'foo')",
		},
		{
			Name: "Equality=Equality",
			Equality: sqlast.Equality(false,
				sqlast.Equality(true,
					sqlast.Bool(true),
					sqlast.Bool(false),
				),
				sqlast.Equality(false,
					sqlast.String("foo"),
					sqlast.String("foo"),
				),
			),
			ExpectedSQL: "(true != false) = ('foo' = 'foo')",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			gen := sqlast.NewSQLGenerator()
			found := tc.Equality.SQLString(gen)
			if tc.ExpectedErrors > 0 {
				require.Equal(t, tc.ExpectedErrors, len(gen.Errors()), "expected number of errors")
			} else {
				require.Equal(t, tc.ExpectedSQL, found, "expected sql")
			}
		})
	}

}
