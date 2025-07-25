package gormparser

import (
	"testing"
	"time"
)

// TestUser 测试用户模型
type TestUser struct {
	ID        uint       `gorm:"primaryKey;autoIncrement;comment:用户ID"`
	Username  string     `gorm:"column:username;size:50;not null;unique;comment:用户名"`
	Email     string     `gorm:"size:100;not null;unique;comment:邮箱"`
	Password  string     `gorm:"size:255;not null;comment:密码"`
	Age       int        `gorm:"comment:年龄"`
	Balance   float64    `gorm:"precision:10;scale:2;default:0.00;comment:余额"`
	IsActive  bool       `gorm:"default:true;comment:是否激活"`
	CreatedAt time.Time  `gorm:"comment:创建时间"`
	UpdatedAt time.Time  `gorm:"comment:更新时间"`
	DeletedAt *time.Time `gorm:"index;comment:删除时间"`
}

func (TestUser) TableName() string {
	return "test_users"
}

func TestParseStruct(t *testing.T) {
	parser := NewParser()

	tableInfo, err := parser.ParseStruct(&TestUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	// 测试表名
	if tableInfo.Name != "test_user" {
		t.Errorf("Expected table name 'test_user', got '%s'", tableInfo.Name)
	}

	// 测试字段数量
	expectedFieldCount := 10
	if len(tableInfo.Fields) != expectedFieldCount {
		t.Errorf("Expected %d fields, got %d", expectedFieldCount, len(tableInfo.Fields))
	}

	// 测试主键
	if len(tableInfo.PrimaryKey) != 1 || tableInfo.PrimaryKey[0] != "id" {
		t.Errorf("Expected primary key [id], got %v", tableInfo.PrimaryKey)
	}

	// 测试具体字段
	tests := []struct {
		fieldName    string
		columnName   string
		fieldType    FieldType
		isPrimaryKey bool
		isAutoIncr   bool
		size         int
		comment      string
	}{
		{"ID", "id", TypeUint, true, true, 0, "用户ID"},
		{"Username", "username", TypeString, false, false, 50, "用户名"},
		{"Email", "email", TypeString, false, false, 100, "邮箱"},
		{"IsActive", "is_active", TypeBool, false, false, 0, "是否激活"},
		{"CreatedAt", "created_at", TypeTime, false, false, 0, "创建时间"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field := tableInfo.FieldMap[tt.fieldName]
			if field == nil {
				t.Fatalf("Field %s not found", tt.fieldName)
			}

			if field.ColumnName != tt.columnName {
				t.Errorf("Expected column name '%s', got '%s'", tt.columnName, field.ColumnName)
			}

			if field.Type != tt.fieldType {
				t.Errorf("Expected field type '%s', got '%s'", tt.fieldType, field.Type)
			}

			if field.IsPrimaryKey != tt.isPrimaryKey {
				t.Errorf("Expected isPrimaryKey %v, got %v", tt.isPrimaryKey, field.IsPrimaryKey)
			}

			if field.IsAutoIncr != tt.isAutoIncr {
				t.Errorf("Expected isAutoIncr %v, got %v", tt.isAutoIncr, field.IsAutoIncr)
			}

			if field.Size != tt.size {
				t.Errorf("Expected size %d, got %d", tt.size, field.Size)
			}

			if field.Comment != tt.comment {
				t.Errorf("Expected comment '%s', got '%s'", tt.comment, field.Comment)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ID", "id"},
		{"Username", "username"},
		{"IsActive", "is_active"},
		{"CreatedAt", "created_at"},
		{"UpdatedAt", "updated_at"},
		{"DeletedAt", "deleted_at"},
		{"UserID", "user_id"},
		{"XMLParser", "xml_parser"},
		{"HTTPResponse", "http_response"},
		{"APIKey", "api_key"},
		{"IOError", "io_error"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

type InvalidModel struct {
	privateField string
}

func TestParseInvalidStruct(t *testing.T) {
	parser := NewParser()

	// 测试非结构体类型
	_, err := parser.ParseStruct("not a struct")
	if err == nil {
		t.Error("Expected error for non-struct type")
	}

	// 测试只有私有字段的结构体
	tableInfo, err := parser.ParseStruct(&InvalidModel{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(tableInfo.Fields) != 0 {
		t.Errorf("Expected 0 fields for struct with only private fields, got %d", len(tableInfo.Fields))
	}
}

// 测试指针字段
type UserWithPointer struct {
	ID    uint    `gorm:"primaryKey"`
	Name  *string `gorm:"size:100"`
	Age   *int
	Email *string `gorm:"unique"`
}

func TestParseStructWithPointers(t *testing.T) {
	parser := NewParser()

	tableInfo, err := parser.ParseStruct(&UserWithPointer{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	// 测试指针字段类型识别
	nameField := tableInfo.FieldMap["Name"]
	if nameField == nil {
		t.Fatal("Name field not found")
	}

	if nameField.Type != TypeString {
		t.Errorf("Expected string type for *string field, got %s", nameField.Type)
	}

	if nameField.Size != 100 {
		t.Errorf("Expected size 100, got %d", nameField.Size)
	}
}

func TestParseGormTags(t *testing.T) {
	type TagTestModel struct {
		Field1 string  `gorm:"column:custom_name;size:50;not null;unique;index"`
		Field2 int     `gorm:"primaryKey;autoIncrement"`
		Field3 string  `gorm:"default:'default_value';comment:测试注释"`
		Field4 float64 `gorm:"precision:10;scale:2"`
		Field5 string  `gorm:"-"` // 应该被跳过
	}

	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&TagTestModel{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	// Field5 应该被跳过
	if len(tableInfo.Fields) != 4 {
		t.Errorf("Expected 4 fields (Field5 should be skipped), got %d", len(tableInfo.Fields))
	}

	// 测试 Field1
	field1 := tableInfo.FieldMap["Field1"]
	if field1 == nil {
		t.Fatal("Field1 not found")
	}
	if field1.ColumnName != "custom_name" {
		t.Errorf("Expected column name 'custom_name', got '%s'", field1.ColumnName)
	}
	if field1.Size != 50 {
		t.Errorf("Expected size 50, got %d", field1.Size)
	}
	if field1.IsNullable {
		t.Error("Expected field to be not nullable")
	}
	if !field1.IsUnique {
		t.Error("Expected field to be unique")
	}
	if !field1.IsIndex {
		t.Error("Expected field to have index")
	}

	// 测试 Field2
	field2 := tableInfo.FieldMap["Field2"]
	if field2 == nil {
		t.Fatal("Field2 not found")
	}
	if !field2.IsPrimaryKey {
		t.Error("Expected field to be primary key")
	}
	if !field2.IsAutoIncr {
		t.Error("Expected field to be auto increment")
	}

	// 测试 Field3
	field3 := tableInfo.FieldMap["Field3"]
	if field3 == nil {
		t.Fatal("Field3 not found")
	}
	if field3.DefaultValue != "'default_value'" {
		t.Errorf("Expected default value ''default_value'', got '%s'", field3.DefaultValue)
	}
	if field3.Comment != "测试注释" {
		t.Errorf("Expected comment '测试注释', got '%s'", field3.Comment)
	}

	// 测试 Field4
	field4 := tableInfo.FieldMap["Field4"]
	if field4 == nil {
		t.Fatal("Field4 not found")
	}
	if field4.Precision != 10 {
		t.Errorf("Expected precision 10, got %d", field4.Precision)
	}
	if field4.Scale != 2 {
		t.Errorf("Expected scale 2, got %d", field4.Scale)
	}

	// Field5 应该不存在
	if tableInfo.FieldMap["Field5"] != nil {
		t.Error("Field5 should be skipped but was found")
	}
}

func TestFull(t *testing.T) {
	type FullModel struct {
		ID        uint       `gorm:"primaryKey;autoIncrement;comment:ID"`
		Name      string     `gorm:"size:100;not null;comment:名称"`
		CreatedAt time.Time  `gorm:"comment:创建时间"`
		UpdatedAt time.Time  `gorm:"comment:更新时间"`
		DeletedAt *time.Time `gorm:"index;comment:删除时间"`
	}

	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&FullModel{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	if tableInfo.Name != "full_model" {
		t.Errorf("Expected table name 'full_model', got '%s'", tableInfo.Name)
	}

	if len(tableInfo.Fields) != 5 {
		t.Errorf("Expected 5 fields, got %d", len(tableInfo.Fields))
	}
}
