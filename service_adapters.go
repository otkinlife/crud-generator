package crudgen

import (
	"fmt"
	"github.com/otkinlife/crud-generator/models"
	"github.com/otkinlife/crud-generator/services"
	"github.com/otkinlife/crud-generator/types"
)

// ConfigService wraps the existing config service for the package API
type ConfigService struct {
	internal *services.ConfigService
}

// NewConfigService creates a new config service instance
func NewConfigService(dbManager *DatabaseManager) *ConfigService {
	db, err := dbManager.GetMainDB()
	if err != nil {
		panic(fmt.Sprintf("Failed to get main database: %v", err))
	}
	return &ConfigService{
		internal: services.NewConfigServiceWithConnectionsDB(db),
	}
}

// CreateConfigFromStruct creates a config from the package struct
func (cs *ConfigService) CreateConfigFromStruct(config *TableConfig) error {
	// Convert package struct to internal model
	internalConfig := &models.TableConfiguration{
		Name:                  config.Name,
		DBTableName:           config.TableName,
		ConnectionID:          config.ConnectionID,
		CreateStatement:       config.CreateStatement,
		QueryPagination:       config.QueryPagination,
		QueryDisplayFields:    config.QueryDisplayFields,
		QuerySearchFields:     config.QuerySearchFields,
		QuerySortableFields:   config.QuerySortableFields,
		CreateCreatableFields: config.CreateCreatableFields,
		CreateValidationRules: config.CreateValidationRules,
		CreateDefaultValues:   config.CreateDefaultValues,
		UpdateUpdatableFields: config.UpdateUpdatableFields,
		UpdateValidationRules: config.UpdateValidationRules,
		Description:           config.Description,
		Tags:                  config.Tags,
		IsActive:              config.IsActive,
		Version:               config.Version,
	}

	return cs.internal.CreateConfig(internalConfig)
}

// GetConfigByNameAsStruct retrieves a config by name and converts to package struct
func (cs *ConfigService) GetConfigByNameAsStruct(name string) (*TableConfig, error) {
	internalConfig, err := cs.internal.GetConfigByName(name)
	if err != nil {
		return nil, err
	}

	// Convert internal model to package struct
	return &TableConfig{
		ID:                    internalConfig.ID,
		Name:                  internalConfig.Name,
		TableName:             internalConfig.DBTableName,
		ConnectionID:          internalConfig.ConnectionID,
		CreateStatement:       internalConfig.CreateStatement,
		QueryPagination:       internalConfig.QueryPagination,
		QueryDisplayFields:    internalConfig.QueryDisplayFields,
		QuerySearchFields:     internalConfig.QuerySearchFields,
		QuerySortableFields:   internalConfig.QuerySortableFields,
		CreateCreatableFields: internalConfig.CreateCreatableFields,
		CreateValidationRules: internalConfig.CreateValidationRules,
		CreateDefaultValues:   internalConfig.CreateDefaultValues,
		UpdateUpdatableFields: internalConfig.UpdateUpdatableFields,
		UpdateValidationRules: internalConfig.UpdateValidationRules,
		Description:           internalConfig.Description,
		Tags:                  internalConfig.Tags,
		IsActive:              internalConfig.IsActive,
		Version:               internalConfig.Version,
	}, nil
}

// GetConfigsAsStruct retrieves all configs and converts to package structs
func (cs *ConfigService) GetConfigsAsStruct(connectionID string) ([]*TableConfig, error) {
	internalConfigs, err := cs.internal.GetConfigs(connectionID)
	if err != nil {
		return nil, err
	}

	configs := make([]*TableConfig, len(internalConfigs))
	for i, internalConfig := range internalConfigs {
		configs[i] = &TableConfig{
			ID:                    internalConfig.ID,
			Name:                  internalConfig.Name,
			TableName:             internalConfig.TableName,
			ConnectionID:          internalConfig.ConnectionID,
			CreateStatement:       internalConfig.CreateStatement,
			QueryPagination:       internalConfig.QueryPagination,
			QueryDisplayFields:    internalConfig.QueryDisplayFields,
			QuerySearchFields:     internalConfig.QuerySearchFields,
			QuerySortableFields:   internalConfig.QuerySortableFields,
			CreateCreatableFields: internalConfig.CreateCreatableFields,
			CreateValidationRules: internalConfig.CreateValidationRules,
			CreateDefaultValues:   internalConfig.CreateDefaultValues,
			UpdateUpdatableFields: internalConfig.UpdateUpdatableFields,
			UpdateValidationRules: internalConfig.UpdateValidationRules,
			Description:           internalConfig.Description,
			Tags:                  internalConfig.Tags,
			IsActive:              internalConfig.IsActive,
			Version:               internalConfig.Version,
		}
	}

	return configs, nil
}

// UpdateConfigFromStruct updates a config from the package struct
func (cs *ConfigService) UpdateConfigFromStruct(id uint, config *TableConfig) error {
	internalConfig := &models.TableConfiguration{
		Name:                  config.Name,
		DBTableName:           config.TableName,
		ConnectionID:          config.ConnectionID,
		CreateStatement:       config.CreateStatement,
		QueryPagination:       config.QueryPagination,
		QueryDisplayFields:    config.QueryDisplayFields,
		QuerySearchFields:     config.QuerySearchFields,
		QuerySortableFields:   config.QuerySortableFields,
		CreateCreatableFields: config.CreateCreatableFields,
		CreateValidationRules: config.CreateValidationRules,
		CreateDefaultValues:   config.CreateDefaultValues,
		UpdateUpdatableFields: config.UpdateUpdatableFields,
		UpdateValidationRules: config.UpdateValidationRules,
		Description:           config.Description,
		Tags:                  config.Tags,
		IsActive:              config.IsActive,
		Version:               config.Version,
	}

	return cs.internal.UpdateConfig(id, internalConfig)
}

// DeleteConfig deletes a config by ID
func (cs *ConfigService) DeleteConfig(id uint) error {
	return cs.internal.DeleteConfig(id)
}

// TestConnection tests a database connection
func (cs *ConfigService) TestConnection(connectionID string) error {
	return cs.internal.TestConnection(connectionID)
}

// CRUDService wraps the existing CRUD service for the package API
type CRUDService struct {
	internal *services.CRUDService
}

// NewCRUDService creates a new CRUD service instance
func NewCRUDService(dbManager *DatabaseManager) *CRUDService {
	db, err := dbManager.GetMainDB()
	if err != nil {
		panic(fmt.Sprintf("Failed to get main database: %v", err))
	}
	configService := NewConfigService(dbManager)
	return &CRUDService{
		internal: services.NewCRUDServiceWithDB(db, configService.internal),
	}
}

// List performs a list operation
func (cs *CRUDService) List(configName string, params *QueryParams) (*QueryResult, error) {
	// Convert package params to internal params
	internalParams := &types.QueryParams{
		Page:     params.Page,
		PageSize: params.PageSize,
		Search:   params.Search,
	}

	// Convert sort fields
	if len(params.Sort) > 0 {
		internalSort := make([]types.SortField, len(params.Sort))
		for i, sort := range params.Sort {
			internalSort[i] = types.SortField{
				Field: sort.Field,
				Order: types.SortOrder(sort.Order),
			}
		}
		internalParams.Sort = internalSort
	}

	result, err := cs.internal.List(configName, internalParams)
	if err != nil {
		return nil, err
	}

	// Convert internal result to package result
	return &QueryResult{
		Data:       result.Data,
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}, nil
}

// Create creates a new record
func (cs *CRUDService) Create(configName string, data map[string]interface{}) (*CRUDResult, error) {
	result, err := cs.internal.Create(configName, data)
	if err != nil {
		return &CRUDResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Convert validation errors to error message if any
	var errorMsg string
	if len(result.Errors) > 0 {
		errorMsg = result.Errors[0].Message
	}

	return &CRUDResult{
		Success: result.Success,
		Data:    map[string]interface{}{"id": result.ID},
		Error:   errorMsg,
		Message: "Record created successfully",
	}, nil
}

// Update updates an existing record
func (cs *CRUDService) Update(configName string, id interface{}, data map[string]interface{}) (*CRUDResult, error) {
	result, err := cs.internal.Update(configName, id, data)
	if err != nil {
		return &CRUDResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Convert validation errors to error message if any
	var errorMsg string
	if len(result.Errors) > 0 {
		errorMsg = result.Errors[0].Message
	}

	return &CRUDResult{
		Success: result.Success,
		Data:    map[string]interface{}{"rows_affected": result.RowsAffected},
		Error:   errorMsg,
		Message: "Record updated successfully",
	}, nil
}

// Delete deletes a record
func (cs *CRUDService) Delete(configName string, id interface{}) (*CRUDResult, error) {
	result, err := cs.internal.Delete(configName, id)
	if err != nil {
		return &CRUDResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &CRUDResult{
		Success: result.Success,
		Data:    map[string]interface{}{"rows_affected": result.RowsAffected},
		Message: "Record deleted successfully",
	}, nil
}

// GetDict retrieves dictionary data for a field
func (cs *CRUDService) GetDict(configName, field string) ([]DictItem, error) {
	result, err := cs.internal.GetDict(configName, field)
	if err != nil {
		return nil, err
	}

	dictItems := make([]DictItem, len(result))
	for i, item := range result {
		dictItems[i] = DictItem{
			Value: item.Value,
			Label: item.Label,
		}
	}

	return dictItems, nil
}
