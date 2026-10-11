package api

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidateRequest validates a struct with `validate` tags and returns
// a map of field -> []error messages. Returns nil, nil if no errors.
func ValidateRequest(req interface{}) (map[string][]string, error) {
	err := validate.Struct(req)
	if err == nil {
		return nil, nil
	}

	validationErrors := err.(validator.ValidationErrors)
	errMap := make(map[string][]string, len(validationErrors))
	for _, fe := range validationErrors {
		field := fe.Field()
		// Use JSON tag name if present
		if jsonTag := fe.StructField(); jsonTag != "" {
			// fallback to field name
		}
		msg := fe.Tag()
		errMap[field] = append(errMap[field], msg)
	}
	return errMap, nil
}
