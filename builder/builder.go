package builder

import (
	"fmt"

	"github.com/otkinlife/crud-generator/database"
	"github.com/otkinlife/crud-generator/generator"
	"github.com/otkinlife/crud-generator/parser"
	"github.com/otkinlife/crud-generator/services"
	"github.com/otkinlife/crud-generator/types"
	"github.com/otkinlife/crud-generator/validator"
)

type CRUDBuilder struct {
	configID      uint
	config        *types.Config
	schema        *types.TableSchema
	queryGen      *generator.QueryGenerator
	crudGen       *generator.CRUDGenerator
	validator     *validator.Validator
	dictProvider  *generator.DictProvider
	configService *services.ConfigService
}

func NewCRUDBuilderFromConfig(configID uint) (*CRUDBuilder, error) {
	configService := services.NewConfigService()

	// 获取表配置
	tableConfig, err := configService.GetConfigByID(configID)
	if err != nil {
		return nil, fmt.Errorf("failed to load table configuration: %w", err)
	}

	// 转换为legacy配置格式
	config, err := configService.ConvertToLegacyConfig(tableConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert configuration: %w", err)
	}

	// 获取数据库连接（使用字符串连接ID）
	db, err := database.GetDatabaseManager().GetConnection(tableConfig.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 解析建表语句
	parser := parser.NewPostgreSQLParser()
	schema, err := parser.ParseCreateStatement(config.CreateStatement)
	if err != nil {
		return nil, fmt.Errorf("failed to parse create statement: %w", err)
	}

	// 创建生成器和验证器
	queryGen := generator.NewQueryGenerator(schema, config)
	crudGen := generator.NewCRUDGenerator(schema, config)
	validator := validator.NewValidator(config)
	dictProvider := generator.NewDictProvider(db)

	return &CRUDBuilder{
		configID:      configID,
		config:        config,
		schema:        schema,
		queryGen:      queryGen,
		crudGen:       crudGen,
		validator:     validator,
		dictProvider:  dictProvider,
		configService: configService,
	}, nil
}

func (b *CRUDBuilder) Query(params types.QueryParams) (*types.QueryResult, error) {
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	// 获取数据库连接
	tableConfig, err := b.configService.GetConfigByID(b.configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get table configuration: %w", err)
	}

	db, err := database.GetDatabaseManager().GetConnection(tableConfig.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	query, countQuery, args, err := b.queryGen.GenerateQuery(params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query: %w", err)
	}

	var total int64
	if err := db.Raw(countQuery, args...).Scan(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	var data []map[string]interface{}
	rows, err := db.Raw(query, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		data = append(data, row)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &types.QueryResult{
		Data:       data,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (b *CRUDBuilder) Create(data map[string]interface{}) (*types.CreateResult, error) {
	validationErrors := b.validator.ValidateCreate(data)
	if len(validationErrors) > 0 {
		return &types.CreateResult{
			Success: false,
			Errors:  validationErrors,
		}, nil
	}

	// 获取数据库连接
	tableConfig, err := b.configService.GetConfigByID(b.configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get table configuration: %w", err)
	}

	db, err := database.GetDatabaseManager().GetConnection(tableConfig.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	query, args, err := b.crudGen.GenerateInsert(data)
	if err != nil {
		return nil, fmt.Errorf("failed to generate insert query: %w", err)
	}

	var insertedID interface{}
	if err := db.Raw(query, args...).Scan(&insertedID).Error; err != nil {
		return nil, fmt.Errorf("failed to execute insert: %w", err)
	}

	return &types.CreateResult{
		Success: true,
		ID:      insertedID,
	}, nil
}

func (b *CRUDBuilder) Update(id interface{}, data map[string]interface{}) (*types.UpdateResult, error) {
	validationErrors := b.validator.ValidateUpdate(data)
	if len(validationErrors) > 0 {
		return &types.UpdateResult{
			Success: false,
			Errors:  validationErrors,
		}, nil
	}

	// 获取数据库连接
	tableConfig, err := b.configService.GetConfigByID(b.configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get table configuration: %w", err)
	}

	db, err := database.GetDatabaseManager().GetConnection(tableConfig.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	query, args, err := b.crudGen.GenerateUpdate(id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to generate update query: %w", err)
	}

	result := db.Exec(query, args...)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to execute update: %w", result.Error)
	}

	return &types.UpdateResult{
		Success:      true,
		RowsAffected: result.RowsAffected,
	}, nil
}

func (b *CRUDBuilder) Delete(id interface{}) (*types.DeleteResult, error) {
	// 获取数据库连接
	tableConfig, err := b.configService.GetConfigByID(b.configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get table configuration: %w", err)
	}

	db, err := database.GetDatabaseManager().GetConnection(tableConfig.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	query, args, err := b.crudGen.GenerateDelete(id)
	if err != nil {
		return nil, fmt.Errorf("failed to generate delete query: %w", err)
	}

	result := db.Exec(query, args...)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to execute delete: %w", result.Error)
	}

	return &types.DeleteResult{
		Success:      true,
		RowsAffected: result.RowsAffected,
	}, nil
}

func (b *CRUDBuilder) GetDictValues(dictSource *types.DictSource) ([]types.DictItem, error) {
	return b.dictProvider.GetDictValues(dictSource)
}

func (b *CRUDBuilder) GetTableSchema() *types.TableSchema {
	return b.schema
}

func (b *CRUDBuilder) GetConfig() *types.Config {
	return b.config
}

func (b *CRUDBuilder) GetConfigID() uint {
	return b.configID
}

func (b *CRUDBuilder) RefreshConfig() error {
	// 重新加载配置
	tableConfig, err := b.configService.GetConfigByID(b.configID)
	if err != nil {
		return fmt.Errorf("failed to reload table configuration: %w", err)
	}

	config, err := b.configService.ConvertToLegacyConfig(tableConfig)
	if err != nil {
		return fmt.Errorf("failed to convert configuration: %w", err)
	}

	// 重新解析建表语句
	parser := parser.NewPostgreSQLParser()
	schema, err := parser.ParseCreateStatement(config.CreateStatement)
	if err != nil {
		return fmt.Errorf("failed to parse create statement: %w", err)
	}

	// 更新组件
	b.config = config
	b.schema = schema
	b.queryGen = generator.NewQueryGenerator(schema, config)
	b.crudGen = generator.NewCRUDGenerator(schema, config)
	b.validator = validator.NewValidator(config)

	return nil
}
