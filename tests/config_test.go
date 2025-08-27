package builder_test

import (
	"testing"

	"github.com/otkinlife/crud-generator/config"
	"github.com/otkinlife/crud-generator/types"
)

func TestConfigLoader(t *testing.T) {
	configJSON := `{
		"table_name": "test_table",
		"create_statement": "CREATE TABLE test_table (id SERIAL PRIMARY KEY, name VARCHAR(100) NOT NULL)",
		"query_config": {
			"pagination": true,
			"search_fields": [
				{"field": "name", "type": "fuzzy"}
			],
			"sortable_fields": ["id", "name"]
		},
		"create_config": {
			"validation_rules": {
				"name": "required,min=2,max=100"
			}
		}
	}`

	loader := config.NewLoader()
	cfg, err := loader.LoadFromBytes([]byte(configJSON))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.TableName != "test_table" {
		t.Errorf("Expected table name 'test_table', got '%s'", cfg.TableName)
	}

	if cfg.QueryConfig == nil {
		t.Fatal("QueryConfig should not be nil")
	}

	if !cfg.QueryConfig.Pagination {
		t.Error("Pagination should be enabled")
	}

	if len(cfg.QueryConfig.SearchFields) != 1 {
		t.Errorf("Expected 1 search field, got %d", len(cfg.QueryConfig.SearchFields))
	}

	if cfg.QueryConfig.SearchFields[0].Type != types.SearchTypeFuzzy {
		t.Errorf("Expected fuzzy search type, got %s", cfg.QueryConfig.SearchFields[0].Type)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
	}{
		{
			name: "valid config",
			configJSON: `{
				"table_name": "users",
				"create_statement": "CREATE TABLE users (id SERIAL PRIMARY KEY)"
			}`,
			expectError: false,
		},
		{
			name: "missing table name",
			configJSON: `{
				"create_statement": "CREATE TABLE users (id SERIAL PRIMARY KEY)"
			}`,
			expectError: true,
		},
		{
			name: "missing create statement",
			configJSON: `{
				"table_name": "users"
			}`,
			expectError: true,
		},
		{
			name: "invalid search field",
			configJSON: `{
				"table_name": "users",
				"create_statement": "CREATE TABLE users (id SERIAL PRIMARY KEY)",
				"query_config": {
					"search_fields": [
						{"field": "", "type": "fuzzy"}
					]
				}
			}`,
			expectError: true,
		},
	}

	loader := config.NewLoader()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loader.LoadFromBytes([]byte(tt.configJSON))
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
