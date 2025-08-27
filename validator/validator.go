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
	if v.config.CreateConfig == nil || len(v.config.CreateConfig.ValidationRules) == 0 {
		return nil
	}

	return v.validateData(data, v.config.CreateConfig.ValidationRules)
}

func (v *Validator) ValidateUpdate(data map[string]interface{}) []types.ValidationError {
	if v.config.UpdateConfig == nil || len(v.config.UpdateConfig.ValidationRules) == 0 {
		return nil
	}

	return v.validateData(data, v.config.UpdateConfig.ValidationRules)
}

func (v *Validator) validateData(data map[string]interface{}, rules map[string]string) []types.ValidationError {
	var errors []types.ValidationError

	for fieldName, rule := range rules {
		value, exists := data[fieldName]

		if !exists {
			if strings.Contains(rule, "required") {
				errors = append(errors, types.ValidationError{
					Field:   fieldName,
					Tag:     "required",
					Value:   nil,
					Message: fmt.Sprintf("Field '%s' is required", fieldName),
				})
			}
			continue
		}

		fieldErrors := v.validateField(fieldName, value, rule)
		errors = append(errors, fieldErrors...)
	}

	return errors
}

func (v *Validator) validateField(fieldName string, value interface{}, rule string) []types.ValidationError {
	var errors []types.ValidationError

	//structData := map[string]interface{}{
	//	fieldName: value,
	//}

	structType := reflect.StructOf([]reflect.StructField{
		{
			Name: strings.Title(fieldName),
			Type: reflect.TypeOf(value),
			Tag:  reflect.StructTag(fmt.Sprintf(`validate:"%s"`, rule)),
		},
	})

	structValue := reflect.New(structType).Elem()
	structValue.FieldByName(strings.Title(fieldName)).Set(reflect.ValueOf(value))

	err := v.validator.Struct(structValue.Interface())
	if err != nil {
		for _, validationErr := range err.(validator.ValidationErrors) {
			errors = append(errors, types.ValidationError{
				Field:   fieldName,
				Tag:     validationErr.Tag(),
				Value:   value,
				Message: v.getErrorMessage(fieldName, validationErr),
			})
		}
	}

	return errors
}

func (v *Validator) getErrorMessage(fieldName string, err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("Field '%s' is required", fieldName)
	case "min":
		return fmt.Sprintf("Field '%s' must be at least %s", fieldName, err.Param())
	case "max":
		return fmt.Sprintf("Field '%s' must be at most %s", fieldName, err.Param())
	case "len":
		return fmt.Sprintf("Field '%s' must be exactly %s characters long", fieldName, err.Param())
	case "email":
		return fmt.Sprintf("Field '%s' must be a valid email address", fieldName)
	case "url":
		return fmt.Sprintf("Field '%s' must be a valid URL", fieldName)
	case "numeric":
		return fmt.Sprintf("Field '%s' must be numeric", fieldName)
	case "alpha":
		return fmt.Sprintf("Field '%s' must contain only alphabetic characters", fieldName)
	case "alphanum":
		return fmt.Sprintf("Field '%s' must contain only alphanumeric characters", fieldName)
	case "gte":
		return fmt.Sprintf("Field '%s' must be greater than or equal to %s", fieldName, err.Param())
	case "lte":
		return fmt.Sprintf("Field '%s' must be less than or equal to %s", fieldName, err.Param())
	case "gt":
		return fmt.Sprintf("Field '%s' must be greater than %s", fieldName, err.Param())
	case "lt":
		return fmt.Sprintf("Field '%s' must be less than %s", fieldName, err.Param())
	case "oneof":
		return fmt.Sprintf("Field '%s' must be one of: %s", fieldName, err.Param())
	case "uuid":
		return fmt.Sprintf("Field '%s' must be a valid UUID", fieldName)
	case "json":
		return fmt.Sprintf("Field '%s' must be valid JSON", fieldName)
	default:
		return fmt.Sprintf("Field '%s' validation failed: %s", fieldName, err.Tag())
	}
}
