package generator

import (
	"fmt"
	"strings"

	"github.com/otkinlife/crud-generator/types"
	"gorm.io/gorm"
)

type DictProvider struct {
	db *gorm.DB
}

func NewDictProvider(db *gorm.DB) *DictProvider {
	return &DictProvider{
		db: db,
	}
}

func (d *DictProvider) GetDictValues(dictSource *types.DictSource) ([]types.DictItem, error) {
	if dictSource == nil {
		return nil, fmt.Errorf("dict source cannot be nil")
	}

	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s", dictSource.Field, dictSource.Table)

	if dictSource.Where != "" {
		query += " WHERE " + dictSource.Where
	}

	order := string(dictSource.SortOrder)
	if order == "" {
		order = string(types.SortOrderASC)
	}

	query += fmt.Sprintf(" ORDER BY %s %s", dictSource.Field, order)

	rows, err := d.db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to execute dict query: %w", err)
	}
	defer rows.Close()

	var items []types.DictItem

	for rows.Next() {
		var value interface{}
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("failed to scan dict value: %w", err)
		}

		label := fmt.Sprintf("%v", value)
		if value == nil {
			label = ""
		}

		items = append(items, types.DictItem{
			Value: value,
			Label: label,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dict rows: %w", err)
	}

	return items, nil
}

func (d *DictProvider) GetDictValuesWithLabels(dictSource *types.DictSource, labelField string) ([]types.DictItem, error) {
	if dictSource == nil {
		return nil, fmt.Errorf("dict source cannot be nil")
	}

	if labelField == "" {
		return d.GetDictValues(dictSource)
	}

	query := fmt.Sprintf(
		"SELECT DISTINCT %s, %s FROM %s",
		dictSource.Field,
		labelField,
		dictSource.Table,
	)

	if dictSource.Where != "" {
		query += " WHERE " + dictSource.Where
	}

	order := string(dictSource.SortOrder)
	if order == "" {
		order = string(types.SortOrderASC)
	}

	query += fmt.Sprintf(" ORDER BY %s %s", dictSource.Field, order)

	rows, err := d.db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to execute dict query with labels: %w", err)
	}
	defer rows.Close()

	var items []types.DictItem

	for rows.Next() {
		var value, label interface{}
		if err := rows.Scan(&value, &label); err != nil {
			return nil, fmt.Errorf("failed to scan dict value and label: %w", err)
		}

		labelStr := fmt.Sprintf("%v", label)
		if label == nil {
			labelStr = fmt.Sprintf("%v", value)
		}

		items = append(items, types.DictItem{
			Value: value,
			Label: labelStr,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dict rows with labels: %w", err)
	}

	return items, nil
}

func (d *DictProvider) GetDictValuesForMultipleFields(dictSources map[string]*types.DictSource) (map[string][]types.DictItem, error) {
	result := make(map[string][]types.DictItem)

	for fieldName, dictSource := range dictSources {
		if dictSource == nil {
			continue
		}

		items, err := d.GetDictValues(dictSource)
		if err != nil {
			return nil, fmt.Errorf("failed to get dict values for field %s: %w", fieldName, err)
		}

		result[fieldName] = items
	}

	return result, nil
}

func (d *DictProvider) ValidateDictSource(dictSource *types.DictSource) error {
	if dictSource == nil {
		return fmt.Errorf("dict source cannot be nil")
	}

	if strings.TrimSpace(dictSource.Table) == "" {
		return fmt.Errorf("dict source table cannot be empty")
	}

	if strings.TrimSpace(dictSource.Field) == "" {
		return fmt.Errorf("dict source field cannot be empty")
	}

	tableExistsQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = ?
		)
	`

	var exists bool
	if err := d.db.Raw(tableExistsQuery, dictSource.Table).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check if table exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("table %s does not exist", dictSource.Table)
	}

	columnExistsQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.columns 
			WHERE table_name = ? AND column_name = ?
		)
	`

	if err := d.db.Raw(columnExistsQuery, dictSource.Table, dictSource.Field).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check if column exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("column %s does not exist in table %s", dictSource.Field, dictSource.Table)
	}

	return nil
}
