package services

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/otkinlife/crud-generator/database"
	"github.com/otkinlife/crud-generator/models"
	"github.com/otkinlife/crud-generator/types"
	"gorm.io/gorm"
)

type ConfigService struct {
	db        *gorm.DB
	validator *validator.Validate
	dbManager *database.DatabaseManager
}

func NewConfigService() *ConfigService {
	return &ConfigService{
		db:        database.GetDatabaseManager().GetMainDB(),
		validator: validator.New(),
		dbManager: database.GetDatabaseManager(),
	}
}

func (s *ConfigService) CreateConfig(config *models.TableConfiguration) error {
	if err := s.validator.Struct(config); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// 验证JSON字段
	if err := s.validateJSONFields(config); err != nil {
		return fmt.Errorf("JSON validation failed: %w", err)
	}

	// 检查连接ID是否存在于配置中
	connectionIDs := s.dbManager.GetAllConnectionIDs()
	found := false
	for _, id := range connectionIDs {
		if id == config.ConnectionID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("connection '%s' not found in database configuration", config.ConnectionID)
	}

	return s.db.Create(config).Error
}

func (s *ConfigService) GetConfigs(connectionID string) ([]models.ConfigDetails, error) {
	query := s.db.Model(&models.TableConfiguration{}).
		Select(`
			table_configurations.*,
			'' as connection_name,
			'' as db_type
		`).Where("is_active = ?", true)

	if connectionID != "" {
		query = query.Where("connection_id = ?", connectionID)
	}

	var configs []models.ConfigDetails
	if err := query.Find(&configs).Error; err != nil {
		return nil, err
	}

	// 填充连接信息
	for i := range configs {
		if dbConfig, err := s.dbManager.GetConnectionConfig(configs[i].ConnectionID); err == nil {
			configs[i].ConnectionName = dbConfig.Name
			configs[i].DbType = dbConfig.DbType
		}
	}

	return configs, nil
}

func (s *ConfigService) GetConfigByID(id uint) (*models.TableConfiguration, error) {
	var config models.TableConfiguration
	err := s.db.First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *ConfigService) GetConfigByName(name string) (*models.TableConfiguration, error) {
	var config models.TableConfiguration
	err := s.db.Where("name = ? AND is_active = ?", name, true).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *ConfigService) UpdateConfig(id uint, config *models.TableConfiguration) error {
	if err := s.validator.Struct(config); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := s.validateJSONFields(config); err != nil {
		return fmt.Errorf("JSON validation failed: %w", err)
	}

	// 检查连接ID是否存在于配置中
	connectionIDs := s.dbManager.GetAllConnectionIDs()
	found := false
	for _, connID := range connectionIDs {
		if connID == config.ConnectionID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("connection '%s' not found in database configuration", config.ConnectionID)
	}

	config.ID = id
	config.Version++ // 增加版本号

	result := s.db.Model(&models.TableConfiguration{}).Where("id = ?", id).Updates(config)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("configuration not found")
	}

	return nil
}

func (s *ConfigService) DeleteConfig(id uint) error {
	return s.db.Model(&models.TableConfiguration{}).Where("id = ?", id).Update("is_active", false).Error
}

func (s *ConfigService) GetAvailableConnections() (map[string]*models.DatabaseConfig, error) {
	if err := s.dbManager.LoadConfigs(); err != nil {
		return nil, fmt.Errorf("failed to load database configurations: %w", err)
	}

	result := make(map[string]*models.DatabaseConfig)
	connectionIDs := s.dbManager.GetAllConnectionIDs()

	for _, id := range connectionIDs {
		if config, err := s.dbManager.GetConnectionConfig(id); err == nil {
			result[id] = config
		}
	}

	return result, nil
}

func (s *ConfigService) TestConnection(connectionID string) error {
	return s.dbManager.TestConnection(connectionID)
}

func (s *ConfigService) TestConnectionWithTable(connectionID string, tableName string) error {
	return s.dbManager.TestConnectionWithTable(connectionID, tableName)
}

func (s *ConfigService) TestConfigConnection(configID uint) error {
	// 获取配置信息
	config, err := s.GetConfigByID(configID)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// 使用配置中的连接ID和表名进行测试
	return s.dbManager.TestConnectionWithTable(config.ConnectionID, config.DBTableName)
}

func (s *ConfigService) ConvertToLegacyConfig(config *models.TableConfiguration) (*types.Config, error) {
	legacyConfig := &types.Config{
		TableName:       config.DBTableName,
		CreateStatement: config.CreateStatement,
	}

	// 转换查询配置
	if config.QuerySearchFields != "" || config.QuerySortableFields != "" {
		queryConfig := &types.QueryConfig{
			Pagination: config.QueryPagination,
		}

		if config.QuerySearchFields != "" {
			var searchFields []types.SearchField
			if err := json.Unmarshal([]byte(config.QuerySearchFields), &searchFields); err != nil {
				return nil, fmt.Errorf("failed to parse search fields: %w", err)
			}
			queryConfig.SearchFields = searchFields
		}

		if config.QuerySortableFields != "" {
			var sortableFields []string
			if err := json.Unmarshal([]byte(config.QuerySortableFields), &sortableFields); err != nil {
				return nil, fmt.Errorf("failed to parse sortable fields: %w", err)
			}
			queryConfig.SortableFields = sortableFields
		}

		legacyConfig.QueryConfig = queryConfig
	}

	// 转换创建配置
	if config.CreateCreatableFields != "" || config.CreateDefaultValues != "" {
		createConfig := &types.CreateConfig{}

		if config.CreateCreatableFields != "" {
			var creatableFields []types.CreatableField
			if err := json.Unmarshal([]byte(config.CreateCreatableFields), &creatableFields); err != nil {
				return nil, fmt.Errorf("failed to parse creatable fields: %w", err)
			}
			createConfig.CreatableFields = creatableFields
		}

		if config.CreateDefaultValues != "" {
			var defaultValues []types.DefaultValue
			if err := json.Unmarshal([]byte(config.CreateDefaultValues), &defaultValues); err != nil {
				return nil, fmt.Errorf("failed to parse default values: %w", err)
			}
			createConfig.DefaultValues = defaultValues
		}

		legacyConfig.CreateConfig = createConfig
	}

	// 转换更新配置
	if config.UpdateUpdatableFields != "" {
		updateConfig := &types.UpdateConfig{}

		var updatableFields []types.UpdatableField
		if err := json.Unmarshal([]byte(config.UpdateUpdatableFields), &updatableFields); err != nil {
			return nil, fmt.Errorf("failed to parse updatable fields: %w", err)
		}
		updateConfig.UpdatableFields = updatableFields

		legacyConfig.UpdateConfig = updateConfig
	}

	return legacyConfig, nil
}

func (s *ConfigService) validateJSONFields(config *models.TableConfiguration) error {
	// 验证展示字段JSON
	if config.QueryDisplayFields != "" {
		var displayFields []types.DisplayField
		if err := json.Unmarshal([]byte(config.QueryDisplayFields), &displayFields); err != nil {
			return fmt.Errorf("invalid query_display_fields JSON: %w", err)
		}
	}

	// 验证搜索字段JSON
	if config.QuerySearchFields != "" {
		var searchFields []types.SearchField
		if err := json.Unmarshal([]byte(config.QuerySearchFields), &searchFields); err != nil {
			return fmt.Errorf("invalid query_search_fields JSON: %w", err)
		}
	}

	// 验证排序字段JSON
	if config.QuerySortableFields != "" {
		var sortableFields []string
		if err := json.Unmarshal([]byte(config.QuerySortableFields), &sortableFields); err != nil {
			return fmt.Errorf("invalid query_sortable_fields JSON: %w", err)
		}
	}

	// 验证可创建字段JSON
	if config.CreateCreatableFields != "" {
		var creatableFields []types.CreatableField
		if err := json.Unmarshal([]byte(config.CreateCreatableFields), &creatableFields); err != nil {
			return fmt.Errorf("invalid create_creatable_fields JSON: %w", err)
		}
	}

	// 验证默认值配置JSON
	if config.CreateDefaultValues != "" {
		var defaultValues []types.DefaultValue
		if err := json.Unmarshal([]byte(config.CreateDefaultValues), &defaultValues); err != nil {
			return fmt.Errorf("invalid create_default_values JSON: %w", err)
		}
	}

	// 验证创建验证规则JSON (兼容旧格式)
	if config.CreateValidationRules != "" {
		var validationRules map[string]string
		if err := json.Unmarshal([]byte(config.CreateValidationRules), &validationRules); err != nil {
			return fmt.Errorf("invalid create_validation_rules JSON: %w", err)
		}
	}

	// 验证可更新字段JSON
	if config.UpdateUpdatableFields != "" {
		var updatableFields []types.UpdatableField
		if err := json.Unmarshal([]byte(config.UpdateUpdatableFields), &updatableFields); err != nil {
			// 尝试解析旧格式 ([]string)
			var legacyFields []string
			if err2 := json.Unmarshal([]byte(config.UpdateUpdatableFields), &legacyFields); err2 != nil {
				return fmt.Errorf("invalid update_updatable_fields JSON: %w", err)
			}
		}
	}

	// 验证更新验证规则JSON (兼容旧格式)
	if config.UpdateValidationRules != "" {
		var validationRules map[string]string
		if err := json.Unmarshal([]byte(config.UpdateValidationRules), &validationRules); err != nil {
			return fmt.Errorf("invalid update_validation_rules JSON: %w", err)
		}
	}

	return nil
}
