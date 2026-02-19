package core

import "fmt"

// ErrValidation is returned when input fails field-level validation.
type ErrValidation struct {
	Field   string
	Message string
}

func (e *ErrValidation) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}
