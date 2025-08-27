package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/otkinlife/crud-generator/types"
)

type Validator struct {
	validator *validator.Validate
	config    *types.Config
}

func NewValidator(config *types.Config) *Validator {
	return &Validator{
		validator: validator.New(),
		config:    config,
	}
}

func (v *Validator) ValidateCreate(data map[string]interface{}) []types.ValidationError {
	if v.config.CreateConfig == nil || len(v.config.CreateConfig.CreatableFields) == 0 {
		return nil
	}

	var errors []types.ValidationError
	for _, field := range v.config.CreateConfig.CreatableFields {
		value, exists := data[field.Field]

		// Check required fields
		if field.Required && !exists {
			errors = append(errors, types.ValidationError{
				Field:   field.Field,
				Tag:     "required",
				Value:   nil,
				Message: fmt.Sprintf("Field '%s' is required", field.Field),
			})
			continue
		}

		// Validate field if value exists and validation rules are defined
		if exists && field.Validation != nil {
			fieldErrors := v.validateFieldWithRules(field.Field, value, field.Validation)
			errors = append(errors, fieldErrors...)
		}
	}

	return errors
}

func (v *Validator) ValidateUpdate(data map[string]interface{}) []types.ValidationError {
	if v.config.UpdateConfig == nil || len(v.config.UpdateConfig.UpdatableFields) == 0 {
		return nil
	}

	var errors []types.ValidationError
	for _, field := range v.config.UpdateConfig.UpdatableFields {
		value, exists := data[field.Field]

		// Check required fields
		if field.Required && !exists {
			errors = append(errors, types.ValidationError{
				Field:   field.Field,
				Tag:     "required",
				Value:   nil,
				Message: fmt.Sprintf("Field '%s' is required", field.Field),
			})
			continue
		}

		// Validate field if value exists and validation rules are defined
		if exists && field.Validation != nil {
			fieldErrors := v.validateFieldWithRules(field.Field, value, field.Validation)
			errors = append(errors, fieldErrors...)
		}
	}

	return errors
}

func (v *Validator) validateFieldWithRules(fieldName string, value interface{}, validation *types.FieldValidation) []types.ValidationError {
	var errors []types.ValidationError

	// Convert value to appropriate type for validation
	strValue := fmt.Sprintf("%v", value)

	// Validate MinLength
	if validation.MinLength != nil {
		if len(strValue) < *validation.MinLength {
			errors = append(errors, types.ValidationError{
				Field:   fieldName,
				Tag:     "min_length",
				Value:   value,
				Message: fmt.Sprintf("Field '%s' must be at least %d characters long", fieldName, *validation.MinLength),
			})
		}
	}

	// Validate MaxLength
	if validation.MaxLength != nil {
		if len(strValue) > *validation.MaxLength {
			errors = append(errors, types.ValidationError{
				Field:   fieldName,
				Tag:     "max_length",
				Value:   value,
				Message: fmt.Sprintf("Field '%s' must be at most %d characters long", fieldName, *validation.MaxLength),
			})
		}
	}

	// Validate Min (for numeric values)
	if validation.Min != nil {
		if numValue, ok := value.(float64); ok {
			if int(numValue) < *validation.Min {
				errors = append(errors, types.ValidationError{
					Field:   fieldName,
					Tag:     "min",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at least %d", fieldName, *validation.Min),
				})
			}
		} else if intValue, ok := value.(int); ok {
			if intValue < *validation.Min {
				errors = append(errors, types.ValidationError{
					Field:   fieldName,
					Tag:     "min",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at least %d", fieldName, *validation.Min),
				})
			}
		}
	}

	// Validate Max (for numeric values)
	if validation.Max != nil {
		if numValue, ok := value.(float64); ok {
			if int(numValue) > *validation.Max {
				errors = append(errors, types.ValidationError{
					Field:   fieldName,
					Tag:     "max",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at most %d", fieldName, *validation.Max),
				})
			}
		} else if intValue, ok := value.(int); ok {
			if intValue > *validation.Max {
				errors = append(errors, types.ValidationError{
					Field:   fieldName,
					Tag:     "max",
					Value:   value,
					Message: fmt.Sprintf("Field '%s' must be at most %d", fieldName, *validation.Max),
				})
			}
		}
	}

	// Validate Pattern (regex)
	if validation.Pattern != "" {
		// Use the existing validator library for regex validation
		structType := reflect.StructOf([]reflect.StructField{
			{
				Name: strings.Title(fieldName),
				Type: reflect.TypeOf(value),
				Tag:  reflect.StructTag(fmt.Sprintf(`validate:"regexp=%s"`, validation.Pattern)),
			},
		})

		structValue := reflect.New(structType).Elem()
		structValue.FieldByName(strings.Title(fieldName)).Set(reflect.ValueOf(value))

		err := v.validator.Struct(structValue.Interface())
		if err != nil {
			message := validation.ErrorMessage
			if message == "" {
				message = fmt.Sprintf("Field '%s' does not match required pattern", fieldName)
			}
			errors = append(errors, types.ValidationError{
				Field:   fieldName,
				Tag:     "pattern",
				Value:   value,
				Message: message,
			})
		}
	}

	return errors
}
