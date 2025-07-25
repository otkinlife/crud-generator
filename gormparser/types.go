package gormparser

import (
	"reflect"
)

// FieldType 字段类型枚举
type FieldType string

const (
	TypeString    FieldType = "string"
	TypeInt       FieldType = "int"
	TypeInt64     FieldType = "int64"
	TypeFloat64   FieldType = "float64"
	TypeBool      FieldType = "bool"
	TypeTime      FieldType = "time"
	TypeBytes     FieldType = "bytes"
	TypeUint      FieldType = "uint"
	TypeUint64    FieldType = "uint64"
	TypeInterface FieldType = "interface"
)

// FilterType 筛选类型枚举
type FilterType string

const (
	FilterEqual   FilterType = "="
	FilterIn      FilterType = "in"
	FilterNotIn   FilterType = "not_in"
	FilterLike    FilterType = "like"
	FilterGT      FilterType = ">"
	FilterGTE     FilterType = ">="
	FilterLT      FilterType = "<"
	FilterLTE     FilterType = "<="
	FilterBetween FilterType = "between"
)

// FieldInfo 字段信息
type FieldInfo struct {
	Name         string            // 字段名
	ColumnName   string            // 数据库列名
	Type         FieldType         // 字段类型
	GoType       reflect.Type      // Go原始类型
	IsPrimaryKey bool              // 是否主键
	IsAutoIncr   bool              // 是否自增
	IsNullable   bool              // 是否可为空
	DefaultValue string            // 默认值
	Size         int               // 字段长度
	Precision    int               // 精度
	Scale        int               // 小数位数
	IsUnique     bool              // 是否唯一
	IsIndex      bool              // 是否索引
	Comment      string            // 注释
	Tags         map[string]string // 所有tag信息

	// 新增扩展字段
	Order    int          // 显示顺序
	Filters  []FilterType // 支持的筛选类型
	Sortable bool         // 是否支持排序
}

// TableInfo 表信息
type TableInfo struct {
	Name       string                // 表名
	Schema     string                // 数据库schema
	Fields     []*FieldInfo          // 字段列表
	FieldMap   map[string]*FieldInfo // 字段映射
	PrimaryKey []string              // 主键字段名列表
	Indexes    map[string][]string   // 索引信息
	Comment    string                // 表注释
}

// ListRequest 列表查询请求
type ListRequest struct {
	// 分页参数
	Page     int `json:"page"`      // 页码，从1开始
	PageSize int `json:"page_size"` // 每页大小

	// 筛选参数
	Filters map[string]FilterCondition `json:"filters"` // 筛选条件

	// 排序参数
	Sort []SortCondition `json:"sort"` // 排序条件

	// 字段选择
	Fields []string `json:"fields"` // 需要查询的字段，为空则查询所有
}

// FilterCondition 筛选条件
type FilterCondition struct {
	Type   FilterType    `json:"type"`   // 筛选类型
	Value  interface{}   `json:"value"`  // 筛选值
	Values []interface{} `json:"values"` // 多值筛选（用于in, not_in等）
}

// SortCondition 排序条件
type SortCondition struct {
	Field string `json:"field"` // 排序字段
	Desc  bool   `json:"desc"`  // 是否降序
}

// ListResponse 列表查询响应
type ListResponse struct {
	SQL    string        `json:"sql"`    // 生成的SQL
	Args   []interface{} `json:"args"`   // SQL参数
	Count  int64         `json:"count"`  // 总数（如果需要）
	Fields []string      `json:"fields"` // 查询的字段列表
}

// QueryError 查询错误
type QueryError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e QueryError) Error() string {
	return e.Message
}
