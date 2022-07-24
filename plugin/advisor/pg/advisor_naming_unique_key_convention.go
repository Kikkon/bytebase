package pg

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

var (
	_ advisor.Advisor = (*NamingUKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(advisor.Postgres, advisor.PostgreSQLNamingUKConvention, &NamingUKConventionAdvisor{})
}

// NamingUKConventionAdvisor is the advisor checking for unique key naming convention.
type NamingUKConventionAdvisor struct {
}

// Check checks for unique key naming convention.
func (check *NamingUKConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, maxLength, err := advisor.UnmarshalNamingRulePayloadAsTemplate(ctx.Rule.Type, ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &namingUKConventionChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
		database:     ctx.Database,
	}

	for _, stmtNode := range root {
		ast.Walk(checker, stmtNode)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

type namingUKConventionChecker struct {
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	format       string
	maxLength    int
	templateList []string
	database     *catalog.Database
}

// Visit implements ast.Visitor interface
func (checker *namingUKConventionChecker) Visit(in ast.Node) ast.Visitor {
	indexDataList := checker.getMetaDataList(in)

	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   "Internal error for unique key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingUKConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf(`Unique key in table "%s" mismatches the naming convention, expect %q but found "%s"`, indexData.tableName, regex, indexData.indexName),
			})
		}
		if checker.maxLength > 0 && len(indexData.indexName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingUKConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf(`Unique key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, indexData.indexName, indexData.tableName, checker.maxLength),
			})
		}
	}

	return checker
}

// getMetaDataList returns the list of unique key with metadata.
func (checker *namingUKConventionChecker) getMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.ConstraintList {
			if metadata := checker.getUniqueKeyMetadata(node.Name.Schema, node.Name.Name, constraint); metadata != nil {
				res = append(res, metadata)
			}
		}
		for _, column := range node.ColumnList {
			for _, constraint := range column.ConstraintList {
				if metadata := checker.getUniqueKeyMetadata(node.Name.Schema, node.Name.Name, constraint); metadata != nil {
					res = append(res, metadata)
				}
			}
		}
	case *ast.AddConstraintStmt:
		constraint := node.Constraint
		if metadata := checker.getUniqueKeyMetadata(node.Table.Schema, node.Table.Name, constraint); metadata != nil {
			res = append(res, metadata)
		}
	case *ast.AddColumnListStmt:
		for _, column := range node.ColumnList {
			for _, constraint := range column.ConstraintList {
				if metadata := checker.getUniqueKeyMetadata(node.Table.Schema, node.Table.Name, constraint); metadata != nil {
					res = append(res, metadata)
				}
			}
		}
	case *ast.CreateIndexStmt:
		if node.Index.Unique {
			var columnList []string
			for _, key := range node.Index.KeyList {
				columnList = append(columnList, key.Key)
			}
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  node.Index.Table.Name,
			}
			res = append(res, &indexMetaData{
				indexName: node.Index.Name,
				tableName: node.Index.Table.Name,
				metaData:  metaData,
			})
		}
	case *ast.RenameConstraintStmt:
		tableName, indexList := checker.findIndex(node.Table.Schema, node.Table.Name, node.ConstraintName)
		if len(indexList) != 0 && indexList[0].Unique && !indexList[0].Primary {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: catalog.JoinColumnListForIndex(indexList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			res = append(res, &indexMetaData{
				indexName: node.NewName,
				tableName: node.Table.Name,
				metaData:  metaData,
			})
		}
	case *ast.RenameIndexStmt:
		// "ALTER INDEX name RENAME TO new_name" doesn't take a table name
		tableName, indexList := checker.findIndex(node.Table.Schema, "", node.IndexName)
		if len(indexList) != 0 && indexList[0].Unique && !indexList[0].Primary {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: catalog.JoinColumnListForIndex(indexList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			res = append(res, &indexMetaData{
				indexName: node.NewName,
				tableName: tableName,
				metaData:  metaData,
			})
		}
	}
	return res
}

// getUniqueKeyMetadata returns index metadata of a unique key constraint, nil if other constraints.
func (checker *namingUKConventionChecker) getUniqueKeyMetadata(schemaName string, tableName string, constraint *ast.ConstraintDef) *indexMetaData {
	switch constraint.Type {
	case ast.ConstraintTypeUnique:
		metaData := map[string]string{
			advisor.ColumnListTemplateToken: strings.Join(constraint.KeyList, "_"),
			advisor.TableNameTemplateToken:  tableName,
		}
		return &indexMetaData{
			indexName: constraint.Name,
			tableName: tableName,
			metaData:  metaData,
		}
	case ast.ConstraintTypeUniqueUsingIndex:
		tableName, indexList := checker.findIndex(schemaName, tableName, constraint.IndexName)
		if len(indexList) != 0 {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: catalog.JoinColumnListForIndex(indexList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			return &indexMetaData{
				indexName: constraint.Name,
				tableName: tableName,
				metaData:  metaData,
			}
		}
	}
	return nil
}

// findIndex returns index found in catalogs, nil if not found
func (checker *namingUKConventionChecker) findIndex(schemaName string, tableName string, indexName string) (string, []*catalog.Index) {
	return checker.database.FindIndex(&catalog.IndexFind{
		SchemaName: normalizeSchemaName(schemaName),
		TableName:  tableName,
		IndexName:  indexName,
	})
}