package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/otkinlife/crud-generator/types"
)

type PostgreSQLParser struct{}

func NewPostgreSQLParser() *PostgreSQLParser {
	return &PostgreSQLParser{}
}

func (p *PostgreSQLParser) ParseCreateStatement(createSQL string) (*types.TableSchema, error) {
	createSQL = strings.TrimSpace(createSQL)

	tableNameRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s(]+)`)
	matches := tableNameRegex.FindStringSubmatch(createSQL)
	if len(matches) < 2 {
		return nil, fmt.Errorf("cannot extract table name from CREATE statement")
	}

	tableName := strings.Trim(matches[1], `"`)

	fieldsRegex := regexp.MustCompile(`(?i)\(\s*(.+)\s*\)`)
	fieldsMatches := fieldsRegex.FindStringSubmatch(createSQL)
	if len(fieldsMatches) < 2 {
		return nil, fmt.Errorf("cannot extract fields from CREATE statement")
	}

	fieldsContent := fieldsMatches[1]
	fields, err := p.parseFields(fieldsContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fields: %w", err)
	}

	return &types.TableSchema{
		TableName: tableName,
		Fields:    fields,
	}, nil
}

func (p *PostgreSQLParser) parseFields(fieldsContent string) ([]types.TableField, error) {
	var fields []types.TableField
	var currentField strings.Builder
	var depth int
	var inQuotes bool
	var quoteChar rune

	for _, char := range fieldsContent {
		switch char {
		case '"', '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
			}
			currentField.WriteRune(char)
		case '(':
			if !inQuotes {
				depth++
			}
			currentField.WriteRune(char)
		case ')':
			if !inQuotes {
				depth--
			}
			currentField.WriteRune(char)
		case ',':
			if !inQuotes && depth == 0 {
				fieldStr := strings.TrimSpace(currentField.String())
				if fieldStr != "" && !p.isConstraint(fieldStr) {
					field, err := p.parseField(fieldStr)
					if err != nil {
						return nil, fmt.Errorf("failed to parse field '%s': %w", fieldStr, err)
					}
					fields = append(fields, field)
				}
				currentField.Reset()
			} else {
				currentField.WriteRune(char)
			}
		default:
			currentField.WriteRune(char)
		}
	}

	fieldStr := strings.TrimSpace(currentField.String())
	if fieldStr != "" && !p.isConstraint(fieldStr) {
		field, err := p.parseField(fieldStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field '%s': %w", fieldStr, err)
		}
		fields = append(fields, field)
	}

	return fields, nil
}

func (p *PostgreSQLParser) isConstraint(fieldStr string) bool {
	fieldStr = strings.ToUpper(strings.TrimSpace(fieldStr))
	constraints := []string{"PRIMARY KEY", "FOREIGN KEY", "UNIQUE", "CHECK", "CONSTRAINT"}

	for _, constraint := range constraints {
		if strings.HasPrefix(fieldStr, constraint) {
			return true
		}
	}

	return false
}

func (p *PostgreSQLParser) parseField(fieldStr string) (types.TableField, error) {
	fieldStr = strings.TrimSpace(fieldStr)

	parts := p.splitFieldDefinition(fieldStr)
	if len(parts) < 2 {
		return types.TableField{}, fmt.Errorf("invalid field definition: %s", fieldStr)
	}

	field := types.TableField{
		Name: strings.Trim(parts[0], `"`),
	}

	typeStr := parts[1]

	pgType, length, precision, scale, err := p.parseDataType(typeStr)
	if err != nil {
		return types.TableField{}, fmt.Errorf("failed to parse data type '%s': %w", typeStr, err)
	}

	field.Type = pgType
	field.Length = length
	field.Precision = precision
	field.Scale = scale

	constraintStr := strings.Join(parts[2:], " ")
	field.NotNull = strings.Contains(strings.ToUpper(constraintStr), "NOT NULL")
	field.PrimaryKey = strings.Contains(strings.ToUpper(constraintStr), "PRIMARY KEY")
	field.Unique = strings.Contains(strings.ToUpper(constraintStr), "UNIQUE")

	defaultRegex := regexp.MustCompile(`(?i)DEFAULT\s+([^,\s]+(?:\s+[^,\s]*)*?)(?:\s+(?:NOT\s+NULL|PRIMARY\s+KEY|UNIQUE|CHECK|REFERENCES)|$)`)
	if matches := defaultRegex.FindStringSubmatch(constraintStr); len(matches) > 1 {
		defaultVal := strings.TrimSpace(matches[1])
		field.DefaultValue = &defaultVal
	}

	return field, nil
}

func (p *PostgreSQLParser) splitFieldDefinition(fieldStr string) []string {
	var parts []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune
	var depth int

	for _, char := range fieldStr {
		switch char {
		case '"', '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
			}
			current.WriteRune(char)
		case '(':
			if !inQuotes {
				depth++
			}
			current.WriteRune(char)
		case ')':
			if !inQuotes {
				depth--
			}
			current.WriteRune(char)
		case ' ', '\t', '\n':
			if inQuotes || depth > 0 {
				current.WriteRune(char)
			} else {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func (p *PostgreSQLParser) parseDataType(typeStr string) (types.PostgreSQLType, int, int, int, error) {
	typeStr = strings.ToLower(strings.TrimSpace(typeStr))

	numericRegex := regexp.MustCompile(`^(numeric|decimal)\s*\(\s*(\d+)\s*(?:,\s*(\d+))?\s*\)$`)
	if matches := numericRegex.FindStringSubmatch(typeStr); len(matches) > 1 {
		precision, _ := strconv.Atoi(matches[2])
		scale := 0
		if len(matches) > 3 && matches[3] != "" {
			scale, _ = strconv.Atoi(matches[3])
		}
		return types.PostgreSQLTypeNumeric, 0, precision, scale, nil
	}

	sizedRegex := regexp.MustCompile(`^(varchar|char|bit|varbit)\s*\(\s*(\d+)\s*\)$`)
	if matches := sizedRegex.FindStringSubmatch(typeStr); len(matches) > 1 {
		length, _ := strconv.Atoi(matches[2])
		switch matches[1] {
		case "varchar":
			return types.PostgreSQLTypeVarchar, length, 0, 0, nil
		case "char":
			return types.PostgreSQLTypeChar, length, 0, 0, nil
		}
	}

	arrayRegex := regexp.MustCompile(`^(.+)\[\]$`)
	if matches := arrayRegex.FindStringSubmatch(typeStr); len(matches) > 1 {
		return types.PostgreSQLTypeArray, 0, 0, 0, nil
	}

	typeMap := map[string]types.PostgreSQLType{
		"integer":                     types.PostgreSQLTypeInteger,
		"int":                         types.PostgreSQLTypeInteger,
		"int4":                        types.PostgreSQLTypeInteger,
		"bigint":                      types.PostgreSQLTypeBigint,
		"int8":                        types.PostgreSQLTypeBigint,
		"smallint":                    types.PostgreSQLTypeSmallint,
		"int2":                        types.PostgreSQLTypeSmallint,
		"numeric":                     types.PostgreSQLTypeNumeric,
		"decimal":                     types.PostgreSQLTypeNumeric,
		"real":                        types.PostgreSQLTypeReal,
		"float4":                      types.PostgreSQLTypeReal,
		"double precision":            types.PostgreSQLTypeDouble,
		"float8":                      types.PostgreSQLTypeDouble,
		"text":                        types.PostgreSQLTypeText,
		"varchar":                     types.PostgreSQLTypeVarchar,
		"character varying":           types.PostgreSQLTypeVarchar,
		"char":                        types.PostgreSQLTypeChar,
		"character":                   types.PostgreSQLTypeChar,
		"bytea":                       types.PostgreSQLTypeBytea,
		"boolean":                     types.PostgreSQLTypeBoolean,
		"bool":                        types.PostgreSQLTypeBoolean,
		"date":                        types.PostgreSQLTypeDate,
		"time":                        types.PostgreSQLTypeTime,
		"timestamp":                   types.PostgreSQLTypeTimestamp,
		"timestamp without time zone": types.PostgreSQLTypeTimestamp,
		"timestamptz":                 types.PostgreSQLTypeTimestampTZ,
		"timestamp with time zone":    types.PostgreSQLTypeTimestampTZ,
		"interval":                    types.PostgreSQLTypeInterval,
		"json":                        types.PostgreSQLTypeJSON,
		"jsonb":                       types.PostgreSQLTypeJSONB,
		"uuid":                        types.PostgreSQLTypeUUID,
	}

	if pgType, exists := typeMap[typeStr]; exists {
		return pgType, 0, 0, 0, nil
	}

	return "", 0, 0, 0, fmt.Errorf("unsupported PostgreSQL type: %s", typeStr)
}
