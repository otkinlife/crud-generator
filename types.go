package crudgen

import "time"

// QueryParams represents parameters for querying data
type QueryParams struct {
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Search   map[string]interface{} `json:"search"`
	Sort     []SortField            `json:"sort"`
}

// SortField represents a sort field configuration
type SortField struct {
	Field string    `json:"field"`
	Order SortOrder `json:"order"`
}

// SortOrder represents sort order
type SortOrder string

const (
	SortOrderASC  SortOrder = "asc"
	SortOrderDESC SortOrder = "desc"
)

// QueryResult represents the result of a query operation
type QueryResult struct {
	Data       []map[string]interface{} `json:"data"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

// CRUDResult represents the result of a CRUD operation
type CRUDResult struct {
	Success          bool                   `json:"success"`
	Data             map[string]interface{} `json:"data,omitempty"`
	Error            string                 `json:"error,omitempty"`
	Message          string                 `json:"message,omitempty"`
	ValidationErrors map[string]string      `json:"validation_errors,omitempty"`
}

// DictItem represents a dictionary item for dropdowns
type DictItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// FieldConfig represents configuration for a form field
type FieldConfig struct {
	Field        string `json:"field"`
	Label        string `json:"label"`
	Type         string `json:"type"`
	Required     bool   `json:"required"`
	DefaultType  string `json:"default_type"`
	DefaultValue string `json:"default_value"`
	UserReadonly bool   `json:"user_readonly"`
	Width        *int   `json:"width,omitempty"`
	Sortable     bool   `json:"sortable,omitempty"`
}

// SearchFieldConfig represents configuration for a search field
type SearchFieldConfig struct {
	Field          string `json:"field"`
	Label          string `json:"label"`
	Type           string `json:"type"`
	DictSource     string `json:"dict_source,omitempty"`
	DictSourceType string `json:"dict_source_type,omitempty"`
}

// ValidationRule represents a validation rule for fields
type ValidationRule struct {
	Field string   `json:"field"`
	Rules []string `json:"rules"`
}

// APIResponse represents a standard API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ConfigCreateRequest represents a request to create a new table configuration
type ConfigCreateRequest struct {
	Name            string `json:"name" binding:"required"`
	TableName       string `json:"table_name" binding:"required"`
	ConnectionID    string `json:"connection_id" binding:"required"`
	CreateStatement string `json:"create_statement" binding:"required"`
	Description     string `json:"description"`
	Tags            string `json:"tags"`
}

// ConfigUpdateRequest represents a request to update a table configuration
type ConfigUpdateRequest struct {
	Name            string `json:"name"`
	TableName       string `json:"table_name"`
	ConnectionID    string `json:"connection_id"`
	CreateStatement string `json:"create_statement"`
	Description     string `json:"description"`
	Tags            string `json:"tags"`

	// Field configurations
	QueryDisplayFields    string `json:"query_display_fields"`
	QuerySearchFields     string `json:"query_search_fields"`
	QuerySortableFields   string `json:"query_sortable_fields"`
	CreateCreatableFields string `json:"create_creatable_fields"`
	UpdateUpdatableFields string `json:"update_updatable_fields"`
}

// ConnectionTestRequest represents a request to test a database connection
type ConnectionTestRequest struct {
	TableName string `json:"table_name,omitempty"`
}

// EmbedOptions represents options for embedding the UI
type EmbedOptions struct {
	BasePath      string            `json:"base_path"`
	Title         string            `json:"title"`
	CustomCSS     string            `json:"custom_css"`
	CustomJS      string            `json:"custom_js"`
	HideHeader    bool              `json:"hide_header"`
	HideFooter    bool              `json:"hide_footer"`
	CustomHeaders map[string]string `json:"custom_headers"`
}

// Middleware configuration
type MiddlewareConfig struct {
	EnableLogging  bool     `json:"enable_logging"`
	EnableCORS     bool     `json:"enable_cors"`
	AllowedOrigins []string `json:"allowed_origins"`
}

// DatabaseInfo represents information about a database connection
type DatabaseInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Host       string    `json:"host"`
	Database   string    `json:"database"`
	Connected  bool      `json:"connected"`
	LastPing   time.Time `json:"last_ping"`
	TableCount int       `json:"table_count"`
}

// TableInfo represents information about a database table
type TableInfo struct {
	Name     string       `json:"name"`
	Schema   string       `json:"schema"`
	Columns  []ColumnInfo `json:"columns"`
	RowCount int64        `json:"row_count"`
}

// ColumnInfo represents information about a table column
type ColumnInfo struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	Nullable        bool   `json:"nullable"`
	DefaultValue    string `json:"default_value"`
	IsPrimaryKey    bool   `json:"is_primary_key"`
	IsAutoIncrement bool   `json:"is_auto_increment"`
}

// PackageInfo contains metadata about the package
type PackageInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	License     string `json:"license"`
	Homepage    string `json:"homepage"`
}

// GetPackageInfo returns information about the package
func GetPackageInfo() *PackageInfo {
	return &PackageInfo{
		Name:        "crud-generator",
		Version:     "1.0.0",
		Description: "A flexible CRUD generator for Go applications",
		Author:      "CRUD Generator Team",
		License:     "MIT",
		Homepage:    "https://github.com/your-org/crud-generator",
	}
}
