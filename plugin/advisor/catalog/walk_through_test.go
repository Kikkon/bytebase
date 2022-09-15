package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor/db"
	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

type testData struct {
	origin    *Database
	statement string
	want      *Database
	err       error
}

var (
	one = "1"
)

func TestWalkThrough(t *testing.T) {
	tests := []testData{
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE,
					c int auto_increment NULL COMMENT 'This is a comment',
					d varchar(10) COLLATE utf8mb4_polish_ci,
					KEY idx_a (a),
					INDEX (b, a),
					UNIQUE (b, c, d),
					FULLTEXT (b, d) WITH PARSER ngram INVISIBLE
				)
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						Name: "",
						TableList: []*Table{
							{
								Name: "t",
								ColumnList: []*Column{
									{
										Name:     "a",
										Position: 1,
										Default:  &one,
										Nullable: false,
										Type:     "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										Default:      nil,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
									{
										Name:     "c",
										Position: 3,
										Default:  nil,
										Nullable: true,
										Type:     "int(11)",
										Comment:  "This is a comment",
									},
									{
										Name:      "d",
										Position:  4,
										Default:   nil,
										Nullable:  true,
										Type:      "varchar(10)",
										Collation: "utf8mb4_polish_ci",
									},
								},
								IndexList: []*Index{
									{
										Name:           "PRIMARY",
										ExpressionList: []string{"a"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        true,
										Visible:        true,
									},
									{
										Name:           "b",
										ExpressionList: []string{"b"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "idx_a",
										ExpressionList: []string{"a"},
										Type:           "BTREE",
										Unique:         false,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "b_2",
										ExpressionList: []string{"b", "a"},
										Type:           "BTREE",
										Unique:         false,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "b_3",
										ExpressionList: []string{"b", "c", "d"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "b_4",
										ExpressionList: []string{"b", "d"},
										Type:           "FULLTEXT",
										Unique:         false,
										Primary:        false,
										Visible:        false,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				CREATE TABLE t(
					a int,
					b int,
					PRIMARY KEY (a, b)
				)
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						Name: "",
						TableList: []*Table{
							{
								Name: "t",
								ColumnList: []*Column{
									{
										Name:     "a",
										Position: 1,
										Type:     "int(11)",
										Nullable: false,
									},
									{
										Name:     "b",
										Position: 2,
										Type:     "int(11)",
										Nullable: false,
									},
								},
								IndexList: []*Index{
									{
										Name:           "PRIMARY",
										ExpressionList: []string{"a", "b"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        true,
										Visible:        true,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				CREATE TABLE t1(a int, b int, c int);
				CREATE TABLE t2(a int);
				DROP TABLE t1, t2
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						TableList:     []*Table{},
						ViewList:      []*View{},
						ExtensionList: []*Extension{},
					},
				},
			},
		},
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				DROP TABLE t1, t2
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeTableNotExists,
				Content: "Table `t1` does not exist",
			},
		},
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE,
					c int auto_increment NULL COMMENT 'This is a comment',
					d varchar(10) COLLATE utf8mb4_polish_ci,
					e int,
					KEY idx_a (a),
					INDEX (b, a),
					UNIQUE (b, c, d),
					FULLTEXT (b, d) WITH PARSER ngram INVISIBLE
				);
				ALTER TABLE t COLLATE utf8mb4_0900_ai_ci, ENGINE = INNODB, COMMENT 'This is a table comment';
				ALTER TABLE t ADD COLUMN a1 int AFTER a;
				ALTER TABLE t ADD INDEX idx_a_b (a, b);
				ALTER TABLE t DROP COLUMN c;
				ALTER TABLE t DROP PRIMARY KEY;
				ALTER TABLE t DROP INDEX b_2;
				ALTER TABLE t MODIFY COLUMN b varchar(20) FIRST;
				ALTER TABLE t CHANGE COLUMN d d_copy varchar(10) COLLATE utf8mb4_polish_ci;
				ALTER TABLE t RENAME COLUMN a to a_copy;
				ALTER TABLE t RENAME TO t_copy;
				ALTER TABLE t_copy ALTER COLUMN a_copy DROP DEFAULT;
				ALTER TABLE t_copy RENAME INDEX b TO idx_b;
				ALTER TABLE t_copy ALTER INDEX b_3 INVISIBLE;
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						Name: "",
						TableList: []*Table{
							{
								Name:      "t_copy",
								Collation: "utf8mb4_0900_ai_ci",
								Engine:    "INNODB",
								Comment:   "This is a table comment",
								ColumnList: []*Column{
									{
										Name:     "b",
										Position: 1,
										Default:  nil,
										Nullable: true,
										Type:     "varchar(20)",
									},
									{
										Name:     "a_copy",
										Position: 2,
										Default:  nil,
										Nullable: false,
										Type:     "int(11)",
									},
									{
										Name:     "a1",
										Position: 3,
										Default:  nil,
										Nullable: true,
										Type:     "int(11)",
									},

									{
										Name:      "d_copy",
										Position:  4,
										Default:   nil,
										Nullable:  true,
										Type:      "varchar(10)",
										Collation: "utf8mb4_polish_ci",
									},
									{
										Name:     "e",
										Position: 5,
										Default:  nil,
										Nullable: true,
										Type:     "int(11)",
									},
								},
								IndexList: []*Index{
									{
										Name:           "idx_b",
										ExpressionList: []string{"b"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "idx_a",
										ExpressionList: []string{"a_copy"},
										Type:           "BTREE",
										Unique:         false,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "b_3",
										ExpressionList: []string{"b", "d_copy"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        false,
										Visible:        false,
									},
									{
										Name:           "b_4",
										ExpressionList: []string{"b", "d_copy"},
										Type:           "FULLTEXT",
										Unique:         false,
										Primary:        false,
										Visible:        false,
									},
									{
										Name:           "idx_a_b",
										ExpressionList: []string{"a_copy", "b"},
										Type:           "BTREE",
										Unique:         false,
										Primary:        false,
										Visible:        true,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE,
					c int auto_increment NULL COMMENT 'This is a comment',
					d varchar(10) COLLATE utf8mb4_polish_ci
				);
				CREATE INDEX idx_a on t(a);
				CREATE INDEX b_2 on t(b, a);
				CREATE UNIQUE INDEX b_3 on t(b, c, d);
				CREATE FULLTEXT INDEX b_4 on t(b, d) WITH PARSER ngram INVISIBLE;
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						Name: "",
						TableList: []*Table{
							{
								Name: "t",
								ColumnList: []*Column{
									{
										Name:     "a",
										Position: 1,
										Default:  &one,
										Nullable: false,
										Type:     "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										Default:      nil,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
									{
										Name:     "c",
										Position: 3,
										Default:  nil,
										Nullable: true,
										Type:     "int(11)",
										Comment:  "This is a comment",
									},
									{
										Name:      "d",
										Position:  4,
										Default:   nil,
										Nullable:  true,
										Type:      "varchar(10)",
										Collation: "utf8mb4_polish_ci",
									},
								},
								IndexList: []*Index{
									{
										Name:           "PRIMARY",
										ExpressionList: []string{"a"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        true,
										Visible:        true,
									},
									{
										Name:           "b",
										ExpressionList: []string{"b"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "idx_a",
										ExpressionList: []string{"a"},
										Type:           "BTREE",
										Unique:         false,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "b_2",
										ExpressionList: []string{"b", "a"},
										Type:           "BTREE",
										Unique:         false,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "b_3",
										ExpressionList: []string{"b", "c", "d"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        false,
										Visible:        true,
									},
									{
										Name:           "b_4",
										ExpressionList: []string{"b", "d"},
										Type:           "FULLTEXT",
										Unique:         false,
										Primary:        false,
										Visible:        false,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE
				);
				DROP INDEX b on t;
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						Name: "",
						TableList: []*Table{
							{
								Name: "t",
								ColumnList: []*Column{
									{
										Name:     "a",
										Position: 1,
										Default:  &one,
										Nullable: false,
										Type:     "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										Default:      nil,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
								},
								IndexList: []*Index{
									{
										Name:           "PRIMARY",
										ExpressionList: []string{"a"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        true,
										Visible:        true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		state := newDatabaseState(test.origin, &FinderContext{CheckIntegrity: true})
		err := state.WalkThrough(test.statement)
		if test.err != nil {
			require.Equal(t, err, test.err)
			continue
		}
		require.NoError(t, err)
		want := newDatabaseState(test.want, &FinderContext{CheckIntegrity: true})
		require.Equal(t, want, state, test.statement)
	}
}

func TestWalkThroughForNoCatalog(t *testing.T) {
	tests := []testData{
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				DROP TABLE t1, t2
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						TableList:     []*Table{},
						ViewList:      []*View{},
						ExtensionList: []*Extension{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		state := newDatabaseState(test.origin, &FinderContext{CheckIntegrity: false})
		err := state.WalkThrough(test.statement)
		if test.err != nil {
			require.Equal(t, err, test.err)
			continue
		}
		require.NoError(t, err)
		want := newDatabaseState(test.want, &FinderContext{CheckIntegrity: false})
		require.Equal(t, want, state, test.statement)
	}
}