package rego2sql

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

func Serialize(n *pg_query.Node) (string, error) {
	dsql, err := pg_query.Deparse(&pg_query.ParseResult{
		Version: 0,
		Stmts: []*pg_query.RawStmt{
			{
				Stmt: &pg_query.Node{
					Node: &pg_query.Node_SelectStmt{
						SelectStmt: &pg_query.SelectStmt{
							LimitOption:    pg_query.LimitOption_LIMIT_OPTION_DEFAULT,
							Op:             pg_query.SetOperation_SETOP_NONE,
							DistinctClause: nil,
							IntoClause:     nil,
							TargetList:     nil,
							FromClause:     nil,
							WhereClause:    n,
							GroupClause:    nil,
							GroupDistinct:  false,
							HavingClause:   nil,
							WindowClause:   nil,
							ValuesLists:    nil,
							SortClause:     nil,
							LimitOffset:    nil,
							LimitCount:     nil,
							LockingClause:  nil,
							WithClause:     nil,
							All:            false,
							Larg:           nil,
							Rarg:           nil,
						},
					},
				},
				StmtLocation: 0,
				StmtLen:      0,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("deparse: %w", err)
	}

	withoutSelect := strings.TrimPrefix(dsql, "SELECT WHERE")
	// TODO: It would be best to find the args and pass them in as parameters, not strings.
	return strings.TrimSpace(withoutSelect), nil
}
