package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitStatements_IgnoresSingleQuoteInsideDoubleQuotedString(t *testing.T) {
	stmts := splitStatements(`INSERT INTO demo VALUES ("x;y's"); SELECT 1;`)
	require.Equal(t, []string{
		`INSERT INTO demo VALUES ("x;y's")`,
		`SELECT 1`,
	}, stmts)
}

func TestSplitStatements_IgnoresDoubleQuoteInsideSingleQuotedString(t *testing.T) {
	stmts := splitStatements(`INSERT INTO demo VALUES ('p;"q'); SELECT 1;`)
	require.Equal(t, []string{
		`INSERT INTO demo VALUES ('p;"q')`,
		`SELECT 1`,
	}, stmts)
}
