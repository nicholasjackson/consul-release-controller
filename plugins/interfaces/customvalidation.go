package interfaces

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// validates that a string is a duration
func ValidateDuration(field validator.FieldLevel) bool {
	_, err := time.ParseDuration(field.Field().String())
	return err == nil
}
