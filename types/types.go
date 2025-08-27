package types

type SearchType string

const (
	SearchTypeFuzzy       SearchType = "fuzzy"
	SearchTypeExact       SearchType = "exact"
	SearchTypeMulti       SearchType = "multi"
	SearchTypeSingle      SearchType = "single"
	SearchTypeRange       SearchType = "range"
	SearchTypeMultiSelect SearchType = "multi_select" // 多选
	SearchTypeDateRange   SearchType = "date_range"   // 日期范围
)

type SortOrder string

const (
	SortOrderASC  SortOrder = "ASC"
	SortOrderDESC SortOrder = "DESC"
)

type PostgreSQLType string

const (
	PostgreSQLTypeInteger     PostgreSQLType = "integer"
	PostgreSQLTypeBigint      PostgreSQLType = "bigint"
	PostgreSQLTypeSmallint    PostgreSQLType = "smallint"
	PostgreSQLTypeNumeric     PostgreSQLType = "numeric"
	PostgreSQLTypeReal        PostgreSQLType = "real"
	PostgreSQLTypeDouble      PostgreSQLType = "double precision"
	PostgreSQLTypeText        PostgreSQLType = "text"
	PostgreSQLTypeVarchar     PostgreSQLType = "varchar"
	PostgreSQLTypeChar        PostgreSQLType = "char"
	PostgreSQLTypeBytea       PostgreSQLType = "bytea"
	PostgreSQLTypeBoolean     PostgreSQLType = "boolean"
	PostgreSQLTypeDate        PostgreSQLType = "date"
	PostgreSQLTypeTime        PostgreSQLType = "time"
	PostgreSQLTypeTimestamp   PostgreSQLType = "timestamp"
	PostgreSQLTypeTimestampTZ PostgreSQLType = "timestamptz"
	PostgreSQLTypeInterval    PostgreSQLType = "interval"
	PostgreSQLTypeJSON        PostgreSQLType = "json"
	PostgreSQLTypeJSONB       PostgreSQLType = "jsonb"
	PostgreSQLTypeUUID        PostgreSQLType = "uuid"
	PostgreSQLTypeArray       PostgreSQLType = "array"
)

type SearchField struct {
	Field      string     `json:"field" validate:"required"`
	Type       SearchType `json:"type" validate:"required"`
	DictSource string     `json:"dict_source,omitempty"` // 改为字符串类型，便于前端处理
}

type DisplayField struct {
	Field      string `json:"field" validate:"required"`
	Label      string `json:"label,omitempty"`
	Width      int    `json:"width,omitempty"`
	Sortable   bool   `json:"sortable,omitempty"`
	Searchable bool   `json:"searchable,omitempty"`
}

type CreatableField struct {
	Field        string           `json:"field" validate:"required"`
	Label        string           `json:"label,omitempty"`
	Type         string           `json:"type,omitempty"` // input, select, textarea, date, etc.
	Required     bool             `json:"required,omitempty"`
	DefaultType  string           `json:"default_type,omitempty"`  // fixed, auto_increment, current_time, uuid
	DefaultValue string           `json:"default_value,omitempty"` // For fixed values
	Validation   *FieldValidation `json:"validation,omitempty"`
	Options      []SelectOption   `json:"options,omitempty"` // For select fields
}

type UpdatableField struct {
	Field      string           `json:"field" validate:"required"`
	Label      string           `json:"label,omitempty"`
	Type       string           `json:"type,omitempty"`
	Required   bool             `json:"required,omitempty"`
	Validation *FieldValidation `json:"validation,omitempty"`
	Options    []SelectOption   `json:"options,omitempty"`
}

type DefaultValue struct {
	Type  string      `json:"type"`            // fixed, auto_increment, current_time, uuid, etc.
	Value interface{} `json:"value,omitempty"` // For fixed values
}

type FieldValidation struct {
	MinLength    *int   `json:"min_length,omitempty"`
	MaxLength    *int   `json:"max_length,omitempty"`
	Min          *int   `json:"min,omitempty"`
	Max          *int   `json:"max,omitempty"`
	Pattern      string `json:"pattern,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type DictSource struct {
	Table     string    `json:"table" validate:"required"`
	Field     string    `json:"field" validate:"required"`
	SortOrder SortOrder `json:"sort_order"`
	Where     string    `json:"where,omitempty"`
}

type QueryConfig struct {
	Pagination     bool           `json:"pagination"`
	DisplayFields  []DisplayField `json:"display_fields,omitempty"`
	SearchFields   []SearchField  `json:"search_fields,omitempty"`
	SortableFields []string       `json:"sortable_fields,omitempty"`
}

type CreateConfig struct {
	CreatableFields []CreatableField `json:"creatable_fields,omitempty"`
	DefaultValues   []DefaultValue   `json:"default_values,omitempty"`
}

type UpdateConfig struct {
	UpdatableFields []UpdatableField `json:"updatable_fields,omitempty"`
}

type Config struct {
	TableName       string        `json:"table_name" validate:"required"`
	CreateStatement string        `json:"create_statement" validate:"required"`
	QueryConfig     *QueryConfig  `json:"query_config,omitempty"`
	CreateConfig    *CreateConfig `json:"create_config,omitempty"`
	UpdateConfig    *UpdateConfig `json:"update_config,omitempty"`
}

type QueryParams struct {
	Page     int                    `json:"page,omitempty"`
	PageSize int                    `json:"page_size,omitempty"`
	Search   map[string]interface{} `json:"search,omitempty"`
	Sort     []SortField            `json:"sort,omitempty"`
}

type SortField struct {
	Field string    `json:"field" validate:"required"`
	Order SortOrder `json:"order"`
}

type QueryResult struct {
	Data       []map[string]interface{} `json:"data"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

type TableField struct {
	Name         string         `json:"name"`
	Type         PostgreSQLType `json:"type"`
	Length       int            `json:"length,omitempty"`
	Precision    int            `json:"precision,omitempty"`
	Scale        int            `json:"scale,omitempty"`
	NotNull      bool           `json:"not_null"`
	PrimaryKey   bool           `json:"primary_key"`
	Unique       bool           `json:"unique"`
	DefaultValue *string        `json:"default_value,omitempty"`
	Comment      string         `json:"comment,omitempty"`
}

type TableSchema struct {
	TableName string       `json:"table_name"`
	Fields    []TableField `json:"fields"`
}

type DictItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ValidationError struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
}

type CreateResult struct {
	ID      interface{}       `json:"id,omitempty"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Success bool              `json:"success"`
}

type UpdateResult struct {
	RowsAffected int64             `json:"rows_affected"`
	Errors       []ValidationError `json:"errors,omitempty"`
	Success      bool              `json:"success"`
}

type DeleteResult struct {
	RowsAffected int64 `json:"rows_affected"`
	Success      bool  `json:"success"`
}
