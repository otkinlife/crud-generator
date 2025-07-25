package gormparser

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// ExtendedUser 带扩展标签的用户模型
type ExtendedUser struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" order:"1" json:"id"`
	Username  string    `gorm:"size:50;not null;unique" order:"2" filter:"=,like" sort:"true" json:"username"`
	Email     string    `gorm:"size:100;not null;unique" order:"3" filter:"=,like" sort:"true" json:"email"`
	Password  string    `gorm:"size:255;not null" order:"10" json:"-"` // 密码字段不参与查询
	Age       int       `gorm:"comment:年龄" order:"4" filter:"=,>,>=,<,<=,between" sort:"true" json:"age"`
	Status    int       `gorm:"comment:状态" order:"5" filter:"=,in,not_in" sort:"true" json:"status"`
	Balance   float64   `gorm:"precision:10;scale:2;default:0.00" order:"6" filter:"=,>,>=,<,<=,between" sort:"true" json:"balance"`
	IsActive  bool      `gorm:"default:true" order:"7" filter:"=" sort:"true" json:"is_active"`
	CreatedAt time.Time `gorm:"comment:创建时间" order:"8" filter:"=,>,>=,<,<=,between" sort:"true" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" order:"9" filter:"=,>,>=,<,<=,between" sort:"true" json:"updated_at"`
}

func TestParseExtendedTags(t *testing.T) {
	parser := NewParser()

	tableInfo, err := parser.ParseStruct(&ExtendedUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	// 测试字段排序
	expectedOrder := []string{"ID", "Username", "Email", "Age", "Status", "Balance", "IsActive", "CreatedAt", "UpdatedAt", "Password"}
	for i, field := range tableInfo.Fields {
		if field.Name != expectedOrder[i] {
			t.Errorf("Expected field %d to be '%s', got '%s'", i, expectedOrder[i], field.Name)
		}
	}

	// 测试具体字段的扩展属性
	tests := []struct {
		fieldName string
		order     int
		filters   []FilterType
		sortable  bool
	}{
		{"Username", 2, []FilterType{FilterEqual, FilterLike}, true},
		{"Age", 4, []FilterType{FilterEqual, FilterGT, FilterGTE, FilterLT, FilterLTE, FilterBetween}, true},
		{"Status", 5, []FilterType{FilterEqual, FilterIn, FilterNotIn}, true},
		{"Password", 10, []FilterType{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field := tableInfo.FieldMap[tt.fieldName]
			if field == nil {
				t.Fatalf("Field %s not found", tt.fieldName)
			}

			if field.Order != tt.order {
				t.Errorf("Expected order %d, got %d", tt.order, field.Order)
			}

			if len(field.Filters) != len(tt.filters) {
				t.Errorf("Expected %d filters, got %d", len(tt.filters), len(field.Filters))
			}

			for i, expectedFilter := range tt.filters {
				if field.Filters[i] != expectedFilter {
					t.Errorf("Expected filter %d to be '%s', got '%s'", i, expectedFilter, field.Filters[i])
				}
			}

			if field.Sortable != tt.sortable {
				t.Errorf("Expected sortable %v, got %v", tt.sortable, field.Sortable)
			}
		})
	}
}

func TestListBasic(t *testing.T) {
	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&ExtendedUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	builder := NewTableBuilder(tableInfo)

	// 基本列表查询
	req := &ListRequest{
		Page:     1,
		PageSize: 10,
	}

	resp, err := builder.List(req)
	if err != nil {
		t.Fatalf("Failed to build list query: %v", err)
	}

	expectedSQL := "SELECT id, username, email, age, status, balance, is_active, created_at, updated_at, password FROM extended_user LIMIT 10 OFFSET 0"
	if resp.SQL != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, resp.SQL)
	}

	if len(resp.Args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(resp.Args))
	}
}

func TestListWithFilters(t *testing.T) {
	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&ExtendedUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	builder := NewTableBuilder(tableInfo)

	// 带筛选条件的查询
	req := &ListRequest{
		Page:     1,
		PageSize: 10,
		Filters: map[string]FilterCondition{
			"Username": {
				Type:  FilterLike,
				Value: "%admin%",
			},
			"Age": {
				Type:  FilterGTE,
				Value: 18,
			},
			"Status": {
				Type:   FilterIn,
				Values: []interface{}{1, 2, 3},
			},
		},
	}

	resp, err := builder.List(req)
	if err != nil {
		t.Fatalf("Failed to build list query: %v", err)
	}

	// 验证SQL包含正确的WHERE条件
	expectedConditions := []string{"username LIKE ?", "age >= ?", "status IN (?, ?, ?)"}
	for _, condition := range expectedConditions {
		if !contains(resp.SQL, condition) {
			t.Errorf("Expected SQL to contain '%s', got: %s", condition, resp.SQL)
		}
	}

	// 验证参数
	expectedArgs := []interface{}{"%admin%", 18, 1, 2, 3}
	if len(resp.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(resp.Args))
	}
}

func TestListWithSort(t *testing.T) {
	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&ExtendedUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	builder := NewTableBuilder(tableInfo)

	// 带排序条件的查询
	req := &ListRequest{
		Page:     1,
		PageSize: 10,
		Sort: []SortCondition{
			{Field: "Username", Desc: false},
			{Field: "CreatedAt", Desc: true},
		},
	}

	resp, err := builder.List(req)
	if err != nil {
		t.Fatalf("Failed to build list query: %v", err)
	}

	expectedOrderBy := "ORDER BY username ASC, created_at DESC"
	if !contains(resp.SQL, expectedOrderBy) {
		t.Errorf("Expected SQL to contain '%s', got: %s", expectedOrderBy, resp.SQL)
	}
}

func TestListWithFieldSelection(t *testing.T) {
	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&ExtendedUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	builder := NewTableBuilder(tableInfo)

	// 只查询指定字段
	req := &ListRequest{
		Page:     1,
		PageSize: 10,
		Fields:   []string{"ID", "Username", "Email"},
	}

	resp, err := builder.List(req)
	if err != nil {
		t.Fatalf("Failed to build list query: %v", err)
	}

	expectedFields := []string{"id", "username", "email"}
	if len(resp.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(resp.Fields))
	}

	for i, field := range expectedFields {
		if resp.Fields[i] != field {
			t.Errorf("Expected field %d to be '%s', got '%s'", i, field, resp.Fields[i])
		}
	}

	expectedSelectClause := "SELECT id, username, email FROM"
	if !contains(resp.SQL, expectedSelectClause) {
		t.Errorf("Expected SQL to contain '%s', got: %s", expectedSelectClause, resp.SQL)
	}
}

func TestListComplexQuery(t *testing.T) {
	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&ExtendedUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	builder := NewTableBuilder(tableInfo)

	// 复杂查询：包含筛选、排序、字段选择和分页
	req := &ListRequest{
		Page:     2,
		PageSize: 20,
		Fields:   []string{"ID", "Username", "Email", "Age", "Status"},
		Filters: map[string]FilterCondition{
			"Age": {
				Type:   FilterBetween,
				Values: []interface{}{18, 65},
			},
			"Status": {
				Type:   FilterIn,
				Values: []interface{}{1, 2},
			},
			"Username": {
				Type:  FilterLike,
				Value: "%user%",
			},
		},
		Sort: []SortCondition{
			{Field: "Age", Desc: false},
			{Field: "Username", Desc: true},
		},
	}

	resp, err := builder.List(req)
	if err != nil {
		t.Fatalf("Failed to build list query: %v", err)
	}

	// 验证SELECT字段
	expectedSelectClause := "SELECT id, username, email, age, status FROM"
	if !contains(resp.SQL, expectedSelectClause) {
		t.Errorf("Expected SQL to contain '%s', got: %s", expectedSelectClause, resp.SQL)
	}

	// 验证WHERE条件
	expectedConditions := []string{"age BETWEEN ? AND ?", "status IN (?, ?)", "username LIKE ?"}
	for _, condition := range expectedConditions {
		if !contains(resp.SQL, condition) {
			t.Errorf("Expected SQL to contain '%s', got: %s", condition, resp.SQL)
		}
	}

	// 验证ORDER BY
	expectedOrderBy := "ORDER BY age ASC, username DESC"
	if !contains(resp.SQL, expectedOrderBy) {
		t.Errorf("Expected SQL to contain '%s', got: %s", expectedOrderBy, resp.SQL)
	}

	// 验证LIMIT和OFFSET
	expectedLimitOffset := "LIMIT 20 OFFSET 20"
	if !contains(resp.SQL, expectedLimitOffset) {
		t.Errorf("Expected SQL to contain '%s', got: %s", expectedLimitOffset, resp.SQL)
	}

	// 验证参数
	expectedArgs := []interface{}{18, 65, 1, 2, "%user%"}
	if len(resp.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(resp.Args))
	}
}

func TestListValidation(t *testing.T) {
	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&ExtendedUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	builder := NewTableBuilder(tableInfo)

	// 测试不存在的字段
	t.Run("NonexistentField", func(t *testing.T) {
		req := &ListRequest{
			Page:     1,
			PageSize: 10,
			Fields:   []string{"NonexistentField"},
		}

		_, err := builder.List(req)
		if err == nil {
			t.Error("Expected error for nonexistent field")
		}
	})

	// 测试不支持的筛选类型
	t.Run("UnsupportedFilterType", func(t *testing.T) {
		req := &ListRequest{
			Page:     1,
			PageSize: 10,
			Filters: map[string]FilterCondition{
				"Password": { // Password字段不支持任何筛选
					Type:  FilterEqual,
					Value: "test",
				},
			},
		}

		_, err := builder.List(req)
		if err == nil {
			t.Error("Expected error for unsupported filter type")
		}
	})

	// 测试不支持排序的字段
	t.Run("UnsortableField", func(t *testing.T) {
		req := &ListRequest{
			Page:     1,
			PageSize: 10,
			Sort: []SortCondition{
				{Field: "Password", Desc: false}, // Password字段不支持排序
			},
		}

		_, err := builder.List(req)
		if err == nil {
			t.Error("Expected error for unsortable field")
		}
	})

	// 测试between筛选参数验证
	t.Run("InvalidBetweenFilter", func(t *testing.T) {
		req := &ListRequest{
			Page:     1,
			PageSize: 10,
			Filters: map[string]FilterCondition{
				"Age": {
					Type:   FilterBetween,
					Values: []interface{}{18}, // between需要2个值
				},
			},
		}

		_, err := builder.List(req)
		if err == nil {
			t.Error("Expected error for invalid between filter")
		}
	})

	// 测试in筛选参数验证
	t.Run("InvalidInFilter", func(t *testing.T) {
		req := &ListRequest{
			Page:     1,
			PageSize: 10,
			Filters: map[string]FilterCondition{
				"Status": {
					Type:   FilterIn,
					Values: []interface{}{}, // in需要至少一个值
				},
			},
		}

		_, err := builder.List(req)
		if err == nil {
			t.Error("Expected error for invalid in filter")
		}
	})
}

func TestListPagination(t *testing.T) {
	parser := NewParser()
	tableInfo, err := parser.ParseStruct(&ExtendedUser{})
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	builder := NewTableBuilder(tableInfo)

	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedLimit  int
		expectedOffset int
	}{
		{"FirstPage", 1, 10, 10, 0},
		{"SecondPage", 2, 10, 10, 10},
		{"LargePage", 3, 50, 50, 100},
		{"InvalidPage", 0, 10, 10, 0},      // 应该默认为第1页
		{"InvalidSize", 1, 0, 10, 0},       // 应该默认为10
		{"TooLargeSize", 1, 2000, 1000, 0}, // 应该限制为1000
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ListRequest{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}

			resp, err := builder.List(req)
			if err != nil {
				t.Fatalf("Failed to build list query: %v", err)
			}

			expectedLimitOffset := fmt.Sprintf("LIMIT %d OFFSET %d", tt.expectedLimit, tt.expectedOffset)
			if !contains(resp.SQL, expectedLimitOffset) {
				t.Errorf("Expected SQL to contain '%s', got: %s", expectedLimitOffset, resp.SQL)
			}
		})
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			strings.Contains(s, substr)))
}
