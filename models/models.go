package models

import (
	"time"
)

type DatabaseConfig struct {
	Name             string                 `json:"name"`
	DbType           string                 `json:"db_type" validate:"required,oneof=postgresql mysql"`
	Host             string                 `json:"host" validate:"required"`
	Port             int                    `json:"port" validate:"required,min=1,max=65535"`
	DatabaseName     string                 `json:"database_name" validate:"required"`
	Username         string                 `json:"username" validate:"required"`
	Password         string                 `json:"password" validate:"required"`
	SSLMode          string                 `json:"ssl_mode"`
	ConnectionParams map[string]interface{} `json:"connection_params"`
	MaxOpenConns     int                    `json:"max_open_conns"`
	MaxIdleConns     int                    `json:"max_idle_conns"`
	ConnMaxLifetime  int                    `json:"conn_max_lifetime"`
	Description      string                 `json:"description"`
}

type DatabaseConfigs map[string]*DatabaseConfig

type TableConfiguration struct {
	ID              uint   `json:"id" gorm:"primaryKey"`
	ConnectionID    string `json:"connection_id" gorm:"size:100;not null;index" validate:"required"` // 改为字符串，引用JSON中的key
	Name            string `json:"name" gorm:"size:100;not null" validate:"required,min=2,max=100"`
	DBTableName     string `json:"table_name" gorm:"column:table_name;size:100;not null" validate:"required"`
	CreateStatement string `json:"create_statement" gorm:"type:text;not null" validate:"required"`

	// 查询配置
	QueryPagination     bool   `json:"query_pagination" gorm:"default:true"`
	QueryDisplayFields  string `json:"query_display_fields" gorm:"type:text"`  // 展示字段配置
	QuerySearchFields   string `json:"query_search_fields" gorm:"type:text"`   // 搜索字段配置
	QuerySortableFields string `json:"query_sortable_fields" gorm:"type:text"` // 可排序字段配置

	// 创建配置
	CreateCreatableFields string `json:"create_creatable_fields" gorm:"type:text"` // 可创建字段配置
	CreateValidationRules string `json:"create_validation_rules" gorm:"type:text"` // 创建验证规则
	CreateDefaultValues   string `json:"create_default_values" gorm:"type:text"`   // 默认值配置

	// 更新配置
	UpdateUpdatableFields string `json:"update_updatable_fields" gorm:"type:text"` // 可更新字段配置
	UpdateValidationRules string `json:"update_validation_rules" gorm:"type:text"` // 更新验证规则

	// 其他配置
	OtherRules string `json:"other_rules" gorm:"type:text"` // 用于存储其他规则或配置

	// 元数据
	Description string `json:"description" gorm:"type:text"`
	Tags        string `json:"tags" gorm:"size:255"`
	IsActive    bool   `json:"is_active" gorm:"default:true"`
	Version     int    `json:"version" gorm:"default:1"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TableConfiguration) TableName() string {
	return "table_configurations"
}

type ConfigDetails struct {
	ID                    uint      `json:"id"`
	ConnectionID          string    `json:"connection_id"`
	ConnectionName        string    `json:"connection_name"`
	DbType                string    `json:"db_type"`
	Name                  string    `json:"name"`
	TableName             string    `json:"table_name"`
	CreateStatement       string    `json:"create_statement"`
	QueryPagination       bool      `json:"query_pagination"`
	QueryDisplayFields    string    `json:"query_display_fields"`
	QuerySearchFields     string    `json:"query_search_fields"`
	QuerySortableFields   string    `json:"query_sortable_fields"`
	CreateCreatableFields string    `json:"create_creatable_fields"`
	CreateValidationRules string    `json:"create_validation_rules"`
	CreateDefaultValues   string    `json:"create_default_values"`
	UpdateUpdatableFields string    `json:"update_updatable_fields"`
	UpdateValidationRules string    `json:"update_validation_rules"`
	Description           string    `json:"description"`
	Tags                  string    `json:"tags"`
	IsActive              bool      `json:"is_active"`
	Version               int       `json:"version"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
