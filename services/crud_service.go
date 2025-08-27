package services

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/otkinlife/crud-generator/database"
	"github.com/otkinlife/crud-generator/models"
	"github.com/otkinlife/crud-generator/types"
)

type CRUDService struct {
	configService *ConfigService
	dbManager     *database.DatabaseManager
	validator     *validator.Validate
}

func NewCRUDService() *CRUDService {
	return &CRUDService{
		configService: NewConfigService(),
		dbManager:     database.GetDatabaseManager(),
		validator:     validator.New(),
	}
}

func (s *CRUDService) GetConfigByName(configName string) (*models.TableConfiguration, error) {
	var config models.TableConfiguration
	err := s.dbManager.GetMainDB().Where("name = ? AND is_active = ?", configName, true).First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("configuration '%s' not found: %w", configName, err)
	}
	return &config, nil
}

func (s *CRUDService) List(configName string, params *types.QueryParams) (*types.QueryResult, error) {
	config, err := s.GetConfigByName(configName)
	if err != nil {
		return nil, err
	}

	// 获取对应的数据库连接
	db, err := s.dbManager.GetConnection(config.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 解析展示字段配置
	var displayFields []types.DisplayField
	if config.QueryDisplayFields != "" {
		if err := json.Unmarshal([]byte(config.QueryDisplayFields), &displayFields); err != nil {
			return nil, fmt.Errorf("failed to parse display fields: %w", err)
		}
	}

	// 解析搜索字段配置
	var searchFields []types.SearchField
	if config.QuerySearchFields != "" {
		if err := json.Unmarshal([]byte(config.QuerySearchFields), &searchFields); err != nil {
			return nil, fmt.Errorf("failed to parse search fields: %w", err)
		}
	}

	// 解析排序字段配置
	var sortableFields []string
	if config.QuerySortableFields != "" {
		if err := json.Unmarshal([]byte(config.QuerySortableFields), &sortableFields); err != nil {
			return nil, fmt.Errorf("failed to parse sortable fields: %w", err)
		}
	}

	// 构建查询
	query := db.Table(config.DBTableName)

	// 应用搜索条件
	if params.Search != nil && len(searchFields) > 0 {
		for _, searchField := range searchFields {
			if searchValue, exists := params.Search[searchField.Field]; exists && searchValue != nil {
				switch searchField.Type {
				case types.SearchTypeFuzzy:
					query = query.Where(fmt.Sprintf("%s ILIKE ?", searchField.Field), fmt.Sprintf("%%%v%%", searchValue))
				case types.SearchTypeExact:
					query = query.Where(fmt.Sprintf("%s = ?", searchField.Field), searchValue)
				case types.SearchTypeRange:
					// 处理范围搜索：先尝试直接转换，然后尝试JSON解析
					var rangeMap map[string]interface{}

					if directMap, ok := searchValue.(map[string]interface{}); ok {
						rangeMap = directMap
					} else if jsonStr, ok := searchValue.(string); ok && jsonStr != "" {
						// 如果是字符串，尝试解析JSON
						if err := json.Unmarshal([]byte(jsonStr), &rangeMap); err != nil {
							// JSON解析失败，跳过这个搜索条件
							continue
						}
					}

					if rangeMap != nil {
						if min, exists := rangeMap["min"]; exists && min != nil {
							query = query.Where(fmt.Sprintf("%s >= ?", searchField.Field), min)
						}
						if max, exists := rangeMap["max"]; exists && max != nil {
							query = query.Where(fmt.Sprintf("%s <= ?", searchField.Field), max)
						}
					}
				case types.SearchTypeSingle, types.SearchTypeMulti:
					query = query.Where(fmt.Sprintf("%s = ?", searchField.Field), searchValue)
				case types.SearchTypeMultiSelect:
					// 多选：处理数组值或JSON字符串，使用 IN 查询
					var values []interface{}

					// 首先尝试直接转换为数组
					if directValues, ok := searchValue.([]interface{}); ok && len(directValues) > 0 {
						values = directValues
					} else if jsonStr, ok := searchValue.(string); ok && jsonStr != "" {
						// 如果是字符串，尝试解析JSON
						var parsedValues []string
						if err := json.Unmarshal([]byte(jsonStr), &parsedValues); err == nil {
							// 转换为[]interface{}
							values = make([]interface{}, len(parsedValues))
							for i, v := range parsedValues {
								values[i] = v
							}
						}
					}

					if len(values) > 0 {
						query = query.Where(fmt.Sprintf("%s IN ?", searchField.Field), values)
					}
				case types.SearchTypeDateRange:
					// 日期范围：处理时间戳范围，先尝试直接转换，然后尝试JSON解析
					var rangeMap map[string]interface{}

					if directMap, ok := searchValue.(map[string]interface{}); ok {
						rangeMap = directMap
					} else if jsonStr, ok := searchValue.(string); ok && jsonStr != "" {
						// 如果是字符串，尝试解析JSON
						if err := json.Unmarshal([]byte(jsonStr), &rangeMap); err != nil {
							// JSON解析失败，跳过这个搜索条件
							continue
						}
					}

					if rangeMap != nil {
						if startTimestamp, exists := rangeMap["start"]; exists && startTimestamp != nil {
							query = query.Where(fmt.Sprintf("%s >= ?", searchField.Field), startTimestamp)
						}
						if endTimestamp, exists := rangeMap["end"]; exists && endTimestamp != nil {
							query = query.Where(fmt.Sprintf("%s <= ?", searchField.Field), endTimestamp)
						}
					}
				}
			}
		}
	}

	// 应用排序
	if params.Sort != nil && len(params.Sort) > 0 {
		for _, sortField := range params.Sort {
			// 验证排序字段是否在允许的字段列表中
			if len(sortableFields) > 0 {
				allowed := false
				for _, allowedField := range sortableFields {
					if allowedField == sortField.Field {
						allowed = true
						break
					}
				}
				if !allowed {
					continue // 跳过不允许的排序字段
				}
			}
			order := "ASC"
			if sortField.Order == types.SortOrderDESC {
				order = "DESC"
			}
			query = query.Order(fmt.Sprintf("%s %s", sortField.Field, order))
		}
	}

	result := &types.QueryResult{}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}
	result.Total = total

	// 应用分页
	if config.QueryPagination && params.Page > 0 && params.PageSize > 0 {
		result.Page = params.Page
		result.PageSize = params.PageSize
		result.TotalPages = int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

		offset := (params.Page - 1) * params.PageSize
		query = query.Offset(offset).Limit(params.PageSize)
	} else {
		result.Page = 1
		result.PageSize = int(total)
		result.TotalPages = 1
	}

	// 执行查询
	var data []map[string]interface{}
	if err := query.Find(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}

	result.Data = data
	return result, nil
}

func (s *CRUDService) Create(configName string, data map[string]interface{}) (*types.CreateResult, error) {
	fmt.Printf("=== CRUDService.Create called with configName: %s, data: %v ===\n", configName, data)
	config, err := s.GetConfigByName(configName)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Config loaded: name=%s, table_name=%s, create_creatable_fields length=%d\n", config.Name, config.DBTableName, len(config.CreateCreatableFields))

	// 获取对应的数据库连接
	db, err := s.dbManager.GetConnection(config.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 解析可创建字段配置
	var creatableFields []types.CreatableField
	fmt.Printf("Raw create_creatable_fields: %s\n", config.CreateCreatableFields)
	if config.CreateCreatableFields != "" {
		if err := json.Unmarshal([]byte(config.CreateCreatableFields), &creatableFields); err != nil {
			return nil, fmt.Errorf("failed to parse creatable fields: %w", err)
		}
		fmt.Printf("Parsed %d creatable fields\n", len(creatableFields))
		for i, field := range creatableFields {
			fmt.Printf("Field %d: %s, default_type: %s\n", i, field.Field, field.DefaultType)
		}
	}

	// 解析默认值配置
	var defaultValues []types.DefaultValue
	if config.CreateDefaultValues != "" {
		if err := json.Unmarshal([]byte(config.CreateDefaultValues), &defaultValues); err != nil {
			return nil, fmt.Errorf("failed to parse default values: %w", err)
		}
	}

	// 应用默认值
	fmt.Printf("Applying default values for %d creatable fields\n", len(creatableFields))
	if len(creatableFields) > 0 {
		for _, field := range creatableFields {
			fmt.Printf("Processing field %s with default_type %s\n", field.Field, field.DefaultType)
			if field.DefaultType != "" && field.Field != "" {
				if _, exists := data[field.Field]; !exists {
					fmt.Printf("Field %s not in data, applying default\n", field.Field)
					switch field.DefaultType {
					case "fixed":
						if field.DefaultValue != "" {
							data[field.Field] = field.DefaultValue
						}
					case "auto_increment":
						// 对于auto_increment字段，尝试生成下一个ID
						// 查询当前最大ID值
						var maxID int64
						query := fmt.Sprintf("COALESCE(MAX(%s), 0)", field.Field)
						result := db.Table(config.DBTableName).Select(query).Row()
						if err := result.Scan(&maxID); err != nil {
							fmt.Printf("Error querying max ID for field %s: %v\n", field.Field, err)
							// 如果查询失败，使用时间戳作为fallback
							maxID = 0
						}
						data[field.Field] = maxID + 1
						fmt.Printf("Generated auto-increment ID for field %s: %d\n", field.Field, maxID+1)
					case "current_time":
						data[field.Field] = "now()"
					case "uuid":
						// 可以使用数据库的UUID函数或者Go的UUID库
						data[field.Field] = "gen_random_uuid()"
					}
				} else {
					fmt.Printf("Field %s already exists in data with value: %v\n", field.Field, data[field.Field])
				}
			}
		}
	}

	// 过滤数据，只保留可创建的字段
	if len(creatableFields) > 0 {
		filteredData := make(map[string]interface{})
		for _, field := range creatableFields {
			if value, exists := data[field.Field]; exists {
				filteredData[field.Field] = value
			}
		}
		data = filteredData
	}

	// 执行字段验证
	if len(creatableFields) > 0 {
		validationErrors := []types.ValidationError{}
		for _, field := range creatableFields {
			value, exists := data[field.Field]

			// 检查必填字段
			if field.Required && (!exists || value == nil || value == "") {
				validationErrors = append(validationErrors, types.ValidationError{
					Field:   field.Field,
					Tag:     "required",
					Value:   value,
					Message: fmt.Sprintf("%s is required", field.Label),
				})
			}

			// 执行其他验证
			if exists && field.Validation != nil {
				if err := s.validateFieldValue(field.Field, value, field.Validation); err != nil {
					validationErrors = append(validationErrors, types.ValidationError{
						Field:   field.Field,
						Tag:     "validation",
						Value:   value,
						Message: err.Error(),
					})
				}
			}
		}

		if len(validationErrors) > 0 {
			return &types.CreateResult{
				Success: false,
				Errors:  validationErrors,
			}, nil
		}
	}

	// 执行插入
	result := db.Table(config.DBTableName).Create(&data)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create record: %w", result.Error)
	}

	// 获取插入的ID（如果有的话）
	var id interface{}
	if idValue, exists := data["id"]; exists {
		id = idValue
	}

	return &types.CreateResult{
		Success: true,
		ID:      id,
	}, nil
}

func (s *CRUDService) Update(configName string, id interface{}, data map[string]interface{}) (*types.UpdateResult, error) {
	config, err := s.GetConfigByName(configName)
	if err != nil {
		return nil, err
	}

	// 获取对应的数据库连接
	db, err := s.dbManager.GetConnection(config.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 解析可更新字段
	var updatableFields []types.UpdatableField
	if config.UpdateUpdatableFields != "" {
		// 尝试解析新格式（对象数组）
		if err := json.Unmarshal([]byte(config.UpdateUpdatableFields), &updatableFields); err != nil {
			// 如果失败，尝试解析旧格式（字符串数组）
			var legacyFields []string
			if err2 := json.Unmarshal([]byte(config.UpdateUpdatableFields), &legacyFields); err2 != nil {
				return nil, fmt.Errorf("failed to parse updatable fields: %w", err)
			}
			// 转换为新格式
			for _, field := range legacyFields {
				updatableFields = append(updatableFields, types.UpdatableField{
					Field:    field,
					Label:    "",
					Type:     "text",
					Required: false,
				})
			}
		}
	}

	// 过滤数据，只保留可更新的字段
	if len(updatableFields) > 0 {
		filteredData := make(map[string]interface{})
		for _, field := range updatableFields {
			if value, exists := data[field.Field]; exists {
				filteredData[field.Field] = value
			}
		}
		data = filteredData
	}

	// 执行字段验证
	if len(updatableFields) > 0 {
		validationErrors := []types.ValidationError{}
		for _, field := range updatableFields {
			value, exists := data[field.Field]

			// 检查必填字段
			if field.Required && (!exists || value == nil || value == "") {
				validationErrors = append(validationErrors, types.ValidationError{
					Field:   field.Field,
					Tag:     "required",
					Value:   value,
					Message: fmt.Sprintf("%s is required", field.Label),
				})
			}

			// 执行其他验证
			if exists && field.Validation != nil {
				if err := s.validateFieldValue(field.Field, value, field.Validation); err != nil {
					validationErrors = append(validationErrors, types.ValidationError{
						Field:   field.Field,
						Tag:     "validation",
						Value:   value,
						Message: err.Error(),
					})
				}
			}
		}

		if len(validationErrors) > 0 {
			return &types.UpdateResult{
				Success: false,
				Errors:  validationErrors,
			}, nil
		}
	}

	// 执行更新
	result := db.Table(config.DBTableName).Where("id = ?", id).Updates(data)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update record: %w", result.Error)
	}

	return &types.UpdateResult{
		Success:      true,
		RowsAffected: result.RowsAffected,
	}, nil
}

func (s *CRUDService) Delete(configName string, id interface{}) (*types.DeleteResult, error) {
	config, err := s.GetConfigByName(configName)
	if err != nil {
		return nil, err
	}

	// 获取对应的数据库连接
	db, err := s.dbManager.GetConnection(config.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 执行删除
	result := db.Table(config.DBTableName).Where("id = ?", id).Delete(&map[string]interface{}{})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to delete record: %w", result.Error)
	}

	return &types.DeleteResult{
		Success:      true,
		RowsAffected: result.RowsAffected,
	}, nil
}

func (s *CRUDService) GetDict(configName string, field string) ([]types.DictItem, error) {
	config, err := s.GetConfigByName(configName)
	if err != nil {
		return nil, err
	}

	// 解析搜索字段配置找到字典源
	var searchFields []types.SearchField
	if config.QuerySearchFields != "" {
		if err := json.Unmarshal([]byte(config.QuerySearchFields), &searchFields); err != nil {
			return nil, fmt.Errorf("failed to parse search fields: %w", err)
		}
	}

	var dictSource *types.DictSource
	var isCustomDict bool
	var customItems []types.DictItem

	for _, searchField := range searchFields {
		if searchField.Field == field && searchField.DictSource != "" {
			// 尝试解析字典源配置
			var parsedDictSource types.DictSource
			if err := json.Unmarshal([]byte(searchField.DictSource), &parsedDictSource); err == nil {
				// 如果是有效的JSON配置，使用解析后的配置
				dictSource = &parsedDictSource
			} else {
				// 检查是否是多行自定义字典数据
				lines := strings.Split(strings.TrimSpace(searchField.DictSource), "\n")
				if len(lines) > 1 || (len(lines) == 1 && strings.Contains(lines[0], "\n")) {
					// 多行数据，当作自定义字典处理
					isCustomDict = true
					for _, line := range lines {
						line = strings.TrimSpace(line)
						if line != "" {
							customItems = append(customItems, types.DictItem{
								Value: line,
								Label: line,
							})
						}
					}
				} else {
					// 单行且不是JSON，当作简单字符串处理，查询当前表中该字段的去重值
					dictSource = &types.DictSource{
						Table:     config.DBTableName, // 使用当前表
						Field:     field,              // 使用当前字段
						SortOrder: types.SortOrderASC,
					}
				}
			}
			break
		}
	}

	// 如果是自定义字典，直接返回
	if isCustomDict {
		return customItems, nil
	}

	if dictSource == nil {
		return nil, fmt.Errorf("no dictionary source found for field '%s'", field)
	}

	// 获取对应的数据库连接
	db, err := s.dbManager.GetConnection(config.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 构建字典查询
	query := db.Table(dictSource.Table).Select(fmt.Sprintf("DISTINCT %s as value, %s as label", dictSource.Field, dictSource.Field)).Where(fmt.Sprintf("%s IS NOT NULL", dictSource.Field))

	if dictSource.Where != "" {
		query = query.Where(dictSource.Where)
	}

	if dictSource.SortOrder != "" {
		query = query.Order(fmt.Sprintf("%s %s", dictSource.Field, string(dictSource.SortOrder)))
	}

	var items []types.DictItem
	if err := query.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to query dictionary items: %w", err)
	}

	return items, nil
}

func (s *CRUDService) validateData(data map[string]interface{}, rules map[string]string) []types.ValidationError {
	var errors []types.ValidationError

	for field, rule := range rules {
		value, exists := data[field]
		if !exists && strings.Contains(rule, "required") {
			errors = append(errors, types.ValidationError{
				Field:   field,
				Tag:     "required",
				Message: fmt.Sprintf("Field '%s' is required", field),
			})
			continue
		}

		if !exists {
			continue
		}

		// 解析验证规则
		rulesParts := strings.Split(rule, ",")
		for _, rulePart := range rulesParts {
			rulePart = strings.TrimSpace(rulePart)
			if err := s.validateSingleRule(field, value, rulePart); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func (s *CRUDService) validateFieldValue(field string, value interface{}, validation *types.FieldValidation) error {
	if validation == nil {
		return nil
	}

	// 验证字符串长度
	if validation.MinLength != nil || validation.MaxLength != nil {
		if str, ok := value.(string); ok {
			length := len(str)
			if validation.MinLength != nil && length < *validation.MinLength {
				if validation.ErrorMessage != "" {
					return fmt.Errorf(validation.ErrorMessage)
				}
				return fmt.Errorf("field '%s' must be at least %d characters", field, *validation.MinLength)
			}
			if validation.MaxLength != nil && length > *validation.MaxLength {
				if validation.ErrorMessage != "" {
					return fmt.Errorf(validation.ErrorMessage)
				}
				return fmt.Errorf("field '%s' must be at most %d characters", field, *validation.MaxLength)
			}
		}
	}

	// 验证数值范围
	if validation.Min != nil || validation.Max != nil {
		var numValue int
		var err error

		switch v := value.(type) {
		case int:
			numValue = v
		case float64:
			numValue = int(v)
		case string:
			numValue, err = strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("field '%s' must be a valid number", field)
			}
		default:
			return fmt.Errorf("field '%s' must be a number", field)
		}

		if validation.Min != nil && numValue < *validation.Min {
			if validation.ErrorMessage != "" {
				return fmt.Errorf(validation.ErrorMessage)
			}
			return fmt.Errorf("field '%s' must be at least %d", field, *validation.Min)
		}
		if validation.Max != nil && numValue > *validation.Max {
			if validation.ErrorMessage != "" {
				return fmt.Errorf(validation.ErrorMessage)
			}
			return fmt.Errorf("field '%s' must be at most %d", field, *validation.Max)
		}
	}

	// 验证正则表达式
	if validation.Pattern != "" {
		if str, ok := value.(string); ok {
			matched, err := regexp.MatchString(validation.Pattern, str)
			if err != nil {
				return fmt.Errorf("invalid pattern for field '%s'", field)
			}
			if !matched {
				if validation.ErrorMessage != "" {
					return fmt.Errorf(validation.ErrorMessage)
				}
				return fmt.Errorf("field '%s' does not match required pattern", field)
			}
		}
	}

	return nil
}

func (s *CRUDService) validateSingleRule(field string, value interface{}, rule string) *types.ValidationError {
	switch {
	case rule == "required":
		if value == nil || (reflect.ValueOf(value).Kind() == reflect.String && strings.TrimSpace(value.(string)) == "") {
			return &types.ValidationError{
				Field:   field,
				Tag:     "required",
				Value:   value,
				Message: fmt.Sprintf("Field '%s' is required", field),
			}
		}

	case strings.HasPrefix(rule, "min="):
		minStr := strings.TrimPrefix(rule, "min=")
		min, err := strconv.Atoi(minStr)
		if err != nil {
			return nil
		}

		switch v := value.(type) {
		case string:
			if len(v) < min {
				return &types.ValidationError{
					Field:   field,
					Tag:     "min",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at least %d characters long", field, min),
				}
			}
		case int, int64, float64:
			val := reflect.ValueOf(v)
			if val.Kind() == reflect.Float64 && val.Float() < float64(min) {
				return &types.ValidationError{
					Field:   field,
					Tag:     "min",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at least %d", field, min),
				}
			} else if (val.Kind() == reflect.Int || val.Kind() == reflect.Int64) && val.Int() < int64(min) {
				return &types.ValidationError{
					Field:   field,
					Tag:     "min",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at least %d", field, min),
				}
			}
		}

	case strings.HasPrefix(rule, "max="):
		maxStr := strings.TrimPrefix(rule, "max=")
		max, err := strconv.Atoi(maxStr)
		if err != nil {
			return nil
		}

		switch v := value.(type) {
		case string:
			if len(v) > max {
				return &types.ValidationError{
					Field:   field,
					Tag:     "max",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at most %d characters long", field, max),
				}
			}
		case int, int64, float64:
			val := reflect.ValueOf(v)
			if val.Kind() == reflect.Float64 && val.Float() > float64(max) {
				return &types.ValidationError{
					Field:   field,
					Tag:     "max",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at most %d", field, max),
				}
			} else if (val.Kind() == reflect.Int || val.Kind() == reflect.Int64) && val.Int() > int64(max) {
				return &types.ValidationError{
					Field:   field,
					Tag:     "max",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at most %d", field, max),
				}
			}
		}

	case rule == "email":
		if str, ok := value.(string); ok {
			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !emailRegex.MatchString(str) {
				return &types.ValidationError{
					Field:   field,
					Tag:     "email",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be a valid email address", field),
				}
			}
		}
	}

	return nil
}
