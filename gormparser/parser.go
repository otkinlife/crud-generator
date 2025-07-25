package gormparser

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Parser GORM结构体解析器
type Parser struct {
	TagName string // tag名称，默认为"gorm"
}

// NewParser 创建新的解析器
func NewParser() *Parser {
	return &Parser{
		TagName: "gorm",
	}
}

// ParseStruct 解析GORM结构体
func (p *Parser) ParseStruct(model interface{}) (*TableInfo, error) {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model must be a struct or pointer to struct")
	}

	tableInfo := &TableInfo{
		Name:       p.getTableName(t),
		Fields:     make([]*FieldInfo, 0),
		FieldMap:   make(map[string]*FieldInfo),
		PrimaryKey: make([]string, 0),
		Indexes:    make(map[string][]string),
	}

	// 解析所有字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 跳过非导出字段
		if !field.IsExported() {
			continue
		}

		fieldInfo := p.parseField(field)
		if fieldInfo != nil {
			tableInfo.Fields = append(tableInfo.Fields, fieldInfo)
			tableInfo.FieldMap[fieldInfo.Name] = fieldInfo

			// 收集主键
			if fieldInfo.IsPrimaryKey {
				tableInfo.PrimaryKey = append(tableInfo.PrimaryKey, fieldInfo.ColumnName)
			}
		}
	}

	// 根据order字段排序
	p.sortFieldsByOrder(tableInfo.Fields)

	return tableInfo, nil
}

// sortFieldsByOrder 根据order字段排序
func (p *Parser) sortFieldsByOrder(fields []*FieldInfo) {
	// 简单的冒泡排序，按order排序
	for i := 0; i < len(fields); i++ {
		for j := i + 1; j < len(fields); j++ {
			if fields[i].Order > fields[j].Order {
				fields[i], fields[j] = fields[j], fields[i]
			}
		}
	}
}

// getTableName 获取表名
func (p *Parser) getTableName(t reflect.Type) string {
	return toSnakeCase(t.Name())
}

// parseField 解析字段
func (p *Parser) parseField(field reflect.StructField) *FieldInfo {
	gormTag := field.Tag.Get(p.TagName)

	// 如果有"-"标签，跳过该字段
	if gormTag == "-" {
		return nil
	}

	fieldInfo := &FieldInfo{
		Name:       field.Name,
		ColumnName: p.getColumnName(field),
		Type:       p.getFieldType(field.Type),
		GoType:     field.Type,
		Tags:       make(map[string]string),
		Filters:    make([]FilterType, 0),
		Order:      999, // 默认值，未设置order的字段排在最后
	}

	// 解析所有tag
	fieldInfo.Tags["gorm"] = gormTag
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		fieldInfo.Tags["json"] = jsonTag
	}
	if orderTag := field.Tag.Get("order"); orderTag != "" {
		fieldInfo.Tags["order"] = orderTag
	}
	if filterTag := field.Tag.Get("filter"); filterTag != "" {
		fieldInfo.Tags["filter"] = filterTag
	}
	if sortTag := field.Tag.Get("sort"); sortTag != "" {
		fieldInfo.Tags["sort"] = sortTag
	}

	// 解析gorm标签
	p.parseGormTag(fieldInfo, gormTag)

	// 解析扩展标签
	p.parseExtendedTags(fieldInfo)

	return fieldInfo
}

// parseExtendedTags 解析扩展标签
func (p *Parser) parseExtendedTags(fieldInfo *FieldInfo) {
	// 解析order标签
	if orderTag := fieldInfo.Tags["order"]; orderTag != "" {
		if order, err := strconv.Atoi(orderTag); err == nil {
			fieldInfo.Order = order
		}
	}

	// 解析filter标签
	if filterTag := fieldInfo.Tags["filter"]; filterTag != "" {
		filters := strings.Split(filterTag, ",")
		for _, f := range filters {
			f = strings.TrimSpace(f)
			switch f {
			case "=", "eq":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterEqual)
			case "in":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterIn)
			case "not_in":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterNotIn)
			case "like":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterLike)
			case ">", "gt":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterGT)
			case ">=", "gte":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterGTE)
			case "<", "lt":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterLT)
			case "<=", "lte":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterLTE)
			case "between":
				fieldInfo.Filters = append(fieldInfo.Filters, FilterBetween)
			}
		}
	}

	// 解析sort标签
	if sortTag := fieldInfo.Tags["sort"]; sortTag != "" {
		fieldInfo.Sortable = sortTag == "true"
	}
}

// parseGormTag 解析GORM标签
func (p *Parser) parseGormTag(fieldInfo *FieldInfo, gormTag string) {
	if gormTag == "" {
		return
	}

	tags := strings.Split(gormTag, ";")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		parts := strings.SplitN(tag, ":", 2)
		key := strings.TrimSpace(parts[0])
		var value string
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		}

		switch key {
		case "column":
			if value != "" {
				fieldInfo.ColumnName = value
			}
		case "primaryKey", "primary_key":
			fieldInfo.IsPrimaryKey = true
		case "autoIncrement", "auto_increment":
			fieldInfo.IsAutoIncr = true
		case "not null":
			fieldInfo.IsNullable = false
		case "unique":
			fieldInfo.IsUnique = true
		case "index":
			fieldInfo.IsIndex = true
		case "size":
			if size, err := strconv.Atoi(value); err == nil {
				fieldInfo.Size = size
			}
		case "precision":
			if precision, err := strconv.Atoi(value); err == nil {
				fieldInfo.Precision = precision
			}
		case "scale":
			if scale, err := strconv.Atoi(value); err == nil {
				fieldInfo.Scale = scale
			}
		case "default":
			fieldInfo.DefaultValue = value
		case "comment":
			fieldInfo.Comment = value
		}
	}
}

// getColumnName 获取列名
func (p *Parser) getColumnName(field reflect.StructField) string {
	gormTag := field.Tag.Get(p.TagName)

	// 解析column标签
	tags := strings.Split(gormTag, ";")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if strings.HasPrefix(tag, "column:") {
			return strings.TrimSpace(strings.TrimPrefix(tag, "column:"))
		}
	}

	// 默认使用字段名的snake_case形式
	return toSnakeCase(field.Name)
}

// getFieldType 获取字段类型
func (p *Parser) getFieldType(t reflect.Type) FieldType {
	// 处理指针类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return TypeInt
	case reflect.Int64:
		return TypeInt64
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return TypeUint
	case reflect.Uint64:
		return TypeUint64
	case reflect.Float32, reflect.Float64:
		return TypeFloat64
	case reflect.Bool:
		return TypeBool
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return TypeBytes
		}
	}

	// 检查是否是time.Time
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return TypeTime
	}

	return TypeInterface
}

// toSnakeCase 转换为snake_case
func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// 检查是否需要添加下划线
			prev := rune(s[i-1])
			if prev >= 'a' && prev <= 'z' || prev >= '0' && prev <= '9' {
				result.WriteRune('_')
			}
		}

		// 转换为小写
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
