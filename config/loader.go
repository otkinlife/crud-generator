package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/otkinlife/crud-generator/types"
)

type Loader struct {
	validator *validator.Validate
}

func NewLoader() *Loader {
	return &Loader{
		validator: validator.New(),
	}
}

func (l *Loader) LoadFromFile(configPath string) (*types.Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return l.LoadFromBytes(data)
}

func (l *Loader) LoadFromBytes(data []byte) (*types.Config, error) {
	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := l.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	l.setDefaults(&config)

	return &config, nil
}

func (l *Loader) validateConfig(config *types.Config) error {
	if err := l.validator.Struct(config); err != nil {
		return err
	}

	if config.QueryConfig != nil {
		for _, field := range config.QueryConfig.SearchFields {
			if err := l.validator.Struct(field); err != nil {
				return fmt.Errorf("search field validation failed: %w", err)
			}
			if field.DictSource != nil {
				if err := l.validator.Struct(field.DictSource); err != nil {
					return fmt.Errorf("dict source validation failed: %w", err)
				}
			}
		}
	}

	return nil
}

func (l *Loader) setDefaults(config *types.Config) {
	if config.QueryConfig != nil {
		for i := range config.QueryConfig.SearchFields {
			field := &config.QueryConfig.SearchFields[i]
			if field.DictSource != nil && field.DictSource.SortOrder == "" {
				field.DictSource.SortOrder = types.SortOrderASC
			}
		}
	}
}
