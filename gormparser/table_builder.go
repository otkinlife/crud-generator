package gormparser

import (
	"fmt"
	"strings"
)

// TableBuilder SQL构建器
type TableBuilder struct {
	tableInfo *TableInfo
}

// NewTableBuilder 创建TableBuilder
func NewTableBuilder(tableInfo *TableInfo) *TableBuilder {
	return &TableBuilder{
		tableInfo: tableInfo,
	}
}

// GetTableName 获取表名
func (tb *TableBuilder) GetTableName() string {
	return tb.tableInfo.Name
}

// GetFields 获取所有字段信息
func (tb *TableBuilder) GetFields() []*FieldInfo {
	return tb.tableInfo.Fields
}

// GetField 获取指定字段信息
func (tb *TableBuilder) GetField(name string) *FieldInfo {
	return tb.tableInfo.FieldMap[name]
}

// GetColumnNames 获取所有列名
func (tb *TableBuilder) GetColumnNames() []string {
	columns := make([]string, len(tb.tableInfo.Fields))
	for i, field := range tb.tableInfo.Fields {
		columns[i] = field.ColumnName
	}
	return columns
}

// GetPrimaryKeys 获取主键列名
func (tb *TableBuilder) GetPrimaryKeys() []string {
	return tb.tableInfo.PrimaryKey
}

// List 构建列表查询SQL
func (tb *TableBuilder) List(req *ListRequest) (*ListResponse, error) {
	var errors []QueryError

	// 验证请求参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 1000 {
		req.PageSize = 1000 // 限制最大页面大小
	}

	// 构建字段列表
	fields, err := tb.buildSelectFields(req.Fields)
	if err != nil {
		errors = append(errors, QueryError{Field: "fields", Message: err.Error()})
	}

	// 构建WHERE条件
	whereClause, whereArgs, err := tb.buildWhereClause(req.Filters)
	if err != nil {
		errors = append(errors, QueryError{Field: "filters", Message: err.Error()})
	}

	// 构建ORDER BY条件
	orderClause, err := tb.buildOrderClause(req.Sort)
	if err != nil {
		errors = append(errors, QueryError{Field: "sort", Message: err.Error()})
	}

	if len(errors) > 0 {
		return nil, errors[0] // 返回第一个错误
	}

	// 构建SQL
	var sql strings.Builder
	sql.WriteString("SELECT ")
	sql.WriteString(strings.Join(fields, ", "))
	sql.WriteString(" FROM ")
	sql.WriteString(tb.tableInfo.Name)

	if whereClause != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(whereClause)
	}

	if orderClause != "" {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(orderClause)
	}

	// 添加分页
	offset := (req.Page - 1) * req.PageSize
	sql.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", req.PageSize, offset))

	return &ListResponse{
		SQL:    sql.String(),
		Args:   whereArgs,
		Fields: fields,
	}, nil
}

// buildSelectFields 构建查询字段列表
func (tb *TableBuilder) buildSelectFields(requestFields []string) ([]string, error) {
	if len(requestFields) == 0 {
		// 返回所有字段，按order排序
		fields := make([]string, len(tb.tableInfo.Fields))
		for i, field := range tb.tableInfo.Fields {
			fields[i] = field.ColumnName
		}
		return fields, nil
	}

	// 验证请求的字段是否存在
	var fields []string
	for _, fieldName := range requestFields {
		if field := tb.tableInfo.FieldMap[fieldName]; field != nil {
			fields = append(fields, field.ColumnName)
		} else {
			return nil, fmt.Errorf("field '%s' not found", fieldName)
		}
	}

	return fields, nil
}

// buildWhereClause 构建WHERE条件
func (tb *TableBuilder) buildWhereClause(filters map[string]FilterCondition) (string, []interface{}, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	var conditions []string
	var args []interface{}

	for fieldName, filter := range filters {
		field := tb.tableInfo.FieldMap[fieldName]
		if field == nil {
			return "", nil, fmt.Errorf("field '%s' not found", fieldName)
		}

		// 检查字段是否支持该筛选类型
		if !tb.fieldSupportsFilter(field, filter.Type) {
			return "", nil, fmt.Errorf("field '%s' does not support filter type '%s'", fieldName, filter.Type)
		}

		condition, conditionArgs, err := tb.buildFilterCondition(field, filter)
		if err != nil {
			return "", nil, err
		}

		conditions = append(conditions, condition)
		args = append(args, conditionArgs...)
	}

	return strings.Join(conditions, " AND "), args, nil
}

// fieldSupportsFilter 检查字段是否支持指定的筛选类型
func (tb *TableBuilder) fieldSupportsFilter(field *FieldInfo, filterType FilterType) bool {
	for _, supportedType := range field.Filters {
		if supportedType == filterType {
			return true
		}
	}
	return false
}

// buildFilterCondition 构建单个筛选条件
func (tb *TableBuilder) buildFilterCondition(field *FieldInfo, filter FilterCondition) (string, []interface{}, error) {
	switch filter.Type {
	case FilterEqual:
		return fmt.Sprintf("%s = ?", field.ColumnName), []interface{}{filter.Value}, nil

	case FilterIn:
		if len(filter.Values) == 0 {
			return "", nil, fmt.Errorf("in filter requires values")
		}
		placeholders := make([]string, len(filter.Values))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		return fmt.Sprintf("%s IN (%s)", field.ColumnName, strings.Join(placeholders, ", ")), filter.Values, nil

	case FilterNotIn:
		if len(filter.Values) == 0 {
			return "", nil, fmt.Errorf("not_in filter requires values")
		}
		placeholders := make([]string, len(filter.Values))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		return fmt.Sprintf("%s NOT IN (%s)", field.ColumnName, strings.Join(placeholders, ", ")), filter.Values, nil

	case FilterLike:
		return fmt.Sprintf("%s LIKE ?", field.ColumnName), []interface{}{filter.Value}, nil

	case FilterGT:
		return fmt.Sprintf("%s > ?", field.ColumnName), []interface{}{filter.Value}, nil

	case FilterGTE:
		return fmt.Sprintf("%s >= ?", field.ColumnName), []interface{}{filter.Value}, nil

	case FilterLT:
		return fmt.Sprintf("%s < ?", field.ColumnName), []interface{}{filter.Value}, nil

	case FilterLTE:
		return fmt.Sprintf("%s <= ?", field.ColumnName), []interface{}{filter.Value}, nil

	case FilterBetween:
		if len(filter.Values) != 2 {
			return "", nil, fmt.Errorf("between filter requires exactly 2 values")
		}
		return fmt.Sprintf("%s BETWEEN ? AND ?", field.ColumnName), filter.Values, nil

	default:
		return "", nil, fmt.Errorf("unsupported filter type: %s", filter.Type)
	}
}

// buildOrderClause 构建ORDER BY条件
func (tb *TableBuilder) buildOrderClause(sorts []SortCondition) (string, error) {
	if len(sorts) == 0 {
		return "", nil
	}

	var orderParts []string
	for _, sort := range sorts {
		field := tb.tableInfo.FieldMap[sort.Field]
		if field == nil {
			return "", fmt.Errorf("field '%s' not found", sort.Field)
		}

		if !field.Sortable {
			return "", fmt.Errorf("field '%s' is not sortable", sort.Field)
		}

		orderPart := field.ColumnName
		if sort.Desc {
			orderPart += " DESC"
		} else {
			orderPart += " ASC"
		}

		orderParts = append(orderParts, orderPart)
	}

	return strings.Join(orderParts, ", "), nil
}

// BuildSelectSQL 构建SELECT SQL
func (tb *TableBuilder) BuildSelectSQL(columns ...string) string {
	if len(columns) == 0 {
		columns = tb.GetColumnNames()
	}

	return fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(columns, ", "),
		tb.tableInfo.Name)
}

// BuildInsertSQL 构建INSERT SQL
func (tb *TableBuilder) BuildInsertSQL(excludeAutoIncr bool) string {
	var columns []string
	var placeholders []string

	for _, field := range tb.tableInfo.Fields {
		if excludeAutoIncr && field.IsAutoIncr {
			continue
		}
		columns = append(columns, field.ColumnName)
		placeholders = append(placeholders, "?")
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tb.tableInfo.Name,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
}

// BuildUpdateSQL 构建UPDATE SQL
func (tb *TableBuilder) BuildUpdateSQL(columns ...string) string {
	if len(columns) == 0 {
		for _, field := range tb.tableInfo.Fields {
			if !field.IsPrimaryKey && !field.IsAutoIncr {
				columns = append(columns, field.ColumnName)
			}
		}
	}

	var setParts []string
	for _, column := range columns {
		setParts = append(setParts, fmt.Sprintf("%s = ?", column))
	}

	var whereParts []string
	for _, pk := range tb.tableInfo.PrimaryKey {
		whereParts = append(whereParts, fmt.Sprintf("%s = ?", pk))
	}

	sql := fmt.Sprintf("UPDATE %s SET %s",
		tb.tableInfo.Name,
		strings.Join(setParts, ", "))

	if len(whereParts) > 0 {
		sql += fmt.Sprintf(" WHERE %s", strings.Join(whereParts, " AND "))
	}

	return sql
}

// BuildDeleteSQL 构建DELETE SQL
func (tb *TableBuilder) BuildDeleteSQL() string {
	var whereParts []string
	for _, pk := range tb.tableInfo.PrimaryKey {
		whereParts = append(whereParts, fmt.Sprintf("%s = ?", pk))
	}

	sql := fmt.Sprintf("DELETE FROM %s", tb.tableInfo.Name)
	if len(whereParts) > 0 {
		sql += fmt.Sprintf(" WHERE %s", strings.Join(whereParts, " AND "))
	}

	return sql
}

// BuildCreateTableSQL 构建CREATE TABLE SQL (基础版本)
func (tb *TableBuilder) BuildCreateTableSQL() string {
	var columns []string

	for _, field := range tb.tableInfo.Fields {
		column := tb.buildColumnDefinition(field)
		columns = append(columns, column)
	}

	sql := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)",
		tb.tableInfo.Name,
		strings.Join(columns, ",\n  "))

	return sql
}

// buildColumnDefinition 构建列定义
func (tb *TableBuilder) buildColumnDefinition(field *FieldInfo) string {
	var parts []string
	parts = append(parts, field.ColumnName)

	// 类型定义 (这里是简化版本，实际应该根据数据库类型来)
	switch field.Type {
	case TypeString:
		if field.Size > 0 {
			parts = append(parts, fmt.Sprintf("VARCHAR(%d)", field.Size))
		} else {
			parts = append(parts, "VARCHAR(255)")
		}
	case TypeInt:
		parts = append(parts, "INT")
	case TypeUint:
		parts = append(parts, "INT UNSIGNED")
	case TypeInt64:
		parts = append(parts, "BIGINT")
	case TypeUint64:
		parts = append(parts, "BIGINT UNSIGNED")
	case TypeFloat64:
		if field.Precision > 0 && field.Scale > 0 {
			parts = append(parts, fmt.Sprintf("DECIMAL(%d,%d)", field.Precision, field.Scale))
		} else {
			parts = append(parts, "DOUBLE")
		}
	case TypeBool:
		parts = append(parts, "BOOLEAN")
	case TypeTime:
		parts = append(parts, "DATETIME")
	default:
		parts = append(parts, "TEXT")
	}

	if field.IsPrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}

	if field.IsAutoIncr {
		parts = append(parts, "AUTO_INCREMENT")
	}

	if !field.IsNullable {
		parts = append(parts, "NOT NULL")
	}

	if field.IsUnique {
		parts = append(parts, "UNIQUE")
	}

	if field.DefaultValue != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", field.DefaultValue))
	}

	if field.Comment != "" {
		parts = append(parts, fmt.Sprintf("COMMENT '%s'", field.Comment))
	}

	return strings.Join(parts, " ")
}
