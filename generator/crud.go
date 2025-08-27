package generator

import (
	"fmt"
	"strings"

	"github.com/otkinlife/crud-generator/types"
)

type CRUDGenerator struct {
	schema *types.TableSchema
	config *types.Config
}

func NewCRUDGenerator(schema *types.TableSchema, config *types.Config) *CRUDGenerator {
	return &CRUDGenerator{
		schema: schema,
		config: config,
	}
}

func (g *CRUDGenerator) GenerateInsert(data map[string]interface{}) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("no data provided for insert")
	}

	var fields []string
	var placeholders []string
	var values []interface{}
	argIndex := 1

	fieldMap := make(map[string]types.TableField)
	for _, field := range g.schema.Fields {
		fieldMap[field.Name] = field
	}

	for fieldName, value := range data {
		if _, exists := fieldMap[fieldName]; !exists {
			continue
		}

		fields = append(fields, fieldName)
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
		values = append(values, value)
		argIndex++
	}

	if len(fields) == 0 {
		return "", nil, fmt.Errorf("no valid fields found for insert")
	}

	primaryKeyField := g.getPrimaryKeyField()
	var returningClause string
	if primaryKeyField != nil {
		returningClause = fmt.Sprintf(" RETURNING %s", primaryKeyField.Name)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)%s",
		g.schema.TableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
		returningClause,
	)

	return query, values, nil
}

func (g *CRUDGenerator) GenerateUpdate(id interface{}, data map[string]interface{}) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("no data provided for update")
	}

	primaryKeyField := g.getPrimaryKeyField()
	if primaryKeyField == nil {
		return "", nil, fmt.Errorf("no primary key field found")
	}

	var setParts []string
	var values []interface{}
	argIndex := 1

	fieldMap := make(map[string]types.TableField)
	for _, field := range g.schema.Fields {
		fieldMap[field.Name] = field
	}

	updatableFields := make(map[string]bool)
	if g.config.UpdateConfig != nil && len(g.config.UpdateConfig.UpdatableFields) > 0 {
		for _, field := range g.config.UpdateConfig.UpdatableFields {
			updatableFields[field] = true
		}
	} else {
		for fieldName := range fieldMap {
			if fieldName != primaryKeyField.Name {
				updatableFields[fieldName] = true
			}
		}
	}

	for fieldName, value := range data {
		if fieldName == primaryKeyField.Name {
			continue
		}

		if _, exists := fieldMap[fieldName]; !exists {
			continue
		}

		if !updatableFields[fieldName] {
			continue
		}

		setParts = append(setParts, fmt.Sprintf("%s = $%d", fieldName, argIndex))
		values = append(values, value)
		argIndex++
	}

	if len(setParts) == 0 {
		return "", nil, fmt.Errorf("no valid fields found for update")
	}

	values = append(values, id)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		g.schema.TableName,
		strings.Join(setParts, ", "),
		primaryKeyField.Name,
		argIndex,
	)

	return query, values, nil
}

func (g *CRUDGenerator) GenerateDelete(id interface{}) (string, []interface{}, error) {
	primaryKeyField := g.getPrimaryKeyField()
	if primaryKeyField == nil {
		return "", nil, fmt.Errorf("no primary key field found")
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = $1",
		g.schema.TableName,
		primaryKeyField.Name,
	)

	return query, []interface{}{id}, nil
}

func (g *CRUDGenerator) getPrimaryKeyField() *types.TableField {
	for _, field := range g.schema.Fields {
		if field.PrimaryKey {
			return &field
		}
	}
	return nil
}
