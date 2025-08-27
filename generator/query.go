package generator

import (
	"fmt"
	"strings"

	"github.com/otkinlife/crud-generator/types"
)

type QueryGenerator struct {
	schema *types.TableSchema
	config *types.Config
}

func NewQueryGenerator(schema *types.TableSchema, config *types.Config) *QueryGenerator {
	return &QueryGenerator{
		schema: schema,
		config: config,
	}
}

func (g *QueryGenerator) GenerateQuery(params types.QueryParams) (string, string, []interface{}, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", g.schema.TableName)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", g.schema.TableName)

	var whereConditions []string
	var args []interface{}
	argIndex := 1

	if params.Search != nil && len(params.Search) > 0 && g.config.QueryConfig != nil {
		conditions, searchArgs, newArgIndex, err := g.buildSearchConditions(params.Search, argIndex)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to build search conditions: %w", err)
		}
		whereConditions = append(whereConditions, conditions...)
		args = append(args, searchArgs...)
		argIndex = newArgIndex
	}

	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = " WHERE " + strings.Join(whereConditions, " AND ")
	}

	fullCountQuery := countQuery + whereClause

	var orderClause string
	if len(params.Sort) > 0 && g.config.QueryConfig != nil {
		//orderClause, err := g.buildOrderClause(params.Sort)
		//if err != nil {
		//	return "", "", nil, fmt.Errorf("failed to build order clause: %w", err)
		//}
	}

	var limitClause string
	if g.config.QueryConfig != nil && g.config.QueryConfig.Pagination {
		offset := (params.Page - 1) * params.PageSize
		limitClause = fmt.Sprintf(" LIMIT %d OFFSET %d", params.PageSize, offset)
	}

	fullQuery := baseQuery + whereClause + orderClause + limitClause

	return fullQuery, fullCountQuery, args, nil
}

func (g *QueryGenerator) buildSearchConditions(search map[string]interface{}, startArgIndex int) ([]string, []interface{}, int, error) {
	var conditions []string
	var args []interface{}
	argIndex := startArgIndex

	searchFieldMap := make(map[string]types.SearchField)
	if g.config.QueryConfig != nil {
		for _, field := range g.config.QueryConfig.SearchFields {
			searchFieldMap[field.Field] = field
		}
	}

	for fieldName, value := range search {
		if value == nil {
			continue
		}

		searchField, exists := searchFieldMap[fieldName]
		if !exists {
			continue
		}

		condition, fieldArgs, newArgIndex, err := g.buildFieldCondition(fieldName, value, searchField, argIndex)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to build condition for field %s: %w", fieldName, err)
		}

		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, fieldArgs...)
			argIndex = newArgIndex
		}
	}

	return conditions, args, argIndex, nil
}

func (g *QueryGenerator) buildFieldCondition(fieldName string, value interface{}, searchField types.SearchField, argIndex int) (string, []interface{}, int, error) {
	var condition string
	var args []interface{}

	switch searchField.Type {
	case types.SearchTypeFuzzy:
		strValue := fmt.Sprintf("%v", value)
		if strValue != "" {
			condition = fmt.Sprintf("%s ILIKE $%d", fieldName, argIndex)
			args = append(args, "%"+strValue+"%")
			argIndex++
		}

	case types.SearchTypeExact:
		condition = fmt.Sprintf("%s = $%d", fieldName, argIndex)
		args = append(args, value)
		argIndex++

	case types.SearchTypeMulti:
		if valueSlice, ok := value.([]interface{}); ok && len(valueSlice) > 0 {
			placeholders := make([]string, len(valueSlice))
			for i, v := range valueSlice {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, v)
				argIndex++
			}
			condition = fmt.Sprintf("%s IN (%s)", fieldName, strings.Join(placeholders, ", "))
		}

	case types.SearchTypeSingle:
		condition = fmt.Sprintf("%s = $%d", fieldName, argIndex)
		args = append(args, value)
		argIndex++

	case types.SearchTypeRange:
		if rangeMap, ok := value.(map[string]interface{}); ok {
			var rangeConds []string

			if minVal, exists := rangeMap["min"]; exists && minVal != nil {
				rangeConds = append(rangeConds, fmt.Sprintf("%s >= $%d", fieldName, argIndex))
				args = append(args, minVal)
				argIndex++
			}

			if maxVal, exists := rangeMap["max"]; exists && maxVal != nil {
				rangeConds = append(rangeConds, fmt.Sprintf("%s <= $%d", fieldName, argIndex))
				args = append(args, maxVal)
				argIndex++
			}

			if len(rangeConds) > 0 {
				condition = strings.Join(rangeConds, " AND ")
			}
		}

	default:
		return "", nil, argIndex, fmt.Errorf("unsupported search type: %s", searchField.Type)
	}

	return condition, args, argIndex, nil
}

func (g *QueryGenerator) buildOrderClause(sorts []types.SortField) (string, error) {
	if len(sorts) == 0 {
		return "", nil
	}

	var orderParts []string
	sortableFields := make(map[string]bool)

	if g.config.QueryConfig != nil {
		for _, field := range g.config.QueryConfig.SortableFields {
			sortableFields[field] = true
		}
	}

	for _, sort := range sorts {
		if !sortableFields[sort.Field] {
			return "", fmt.Errorf("field %s is not sortable", sort.Field)
		}

		order := string(sort.Order)
		if order == "" {
			order = string(types.SortOrderASC)
		}

		if order != string(types.SortOrderASC) && order != string(types.SortOrderDESC) {
			return "", fmt.Errorf("invalid sort order: %s", order)
		}

		orderParts = append(orderParts, fmt.Sprintf("%s %s", sort.Field, order))
	}

	return " ORDER BY " + strings.Join(orderParts, ", "), nil
}
