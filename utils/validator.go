package utils

import (
	"github.com/go-playground/validator/v10"
)

func FormatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		errors[e.Field()] = "Invalid " + e.Tag()
	}
	return errors
}
