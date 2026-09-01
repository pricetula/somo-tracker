package validation

import (
	"github.com/google/uuid"
	"somotracker/backend/internal/xerrors"
)

var ErrInvalidUUID = xerrors.InvalidInput("invalid uuid")

func IsValidUUID(s string) error {
	if _, err := uuid.Parse(s); err != nil {
		return ErrInvalidUUID
	}
	return nil
}

func RequireUUIDs(fields map[string]string) error {
	for name, value := range fields {
		if err := IsValidUUID(value); err != nil {
			return &xerrors.DomainError{
				Code:    string(xerrors.CodeInvalidInput),
				Status:  400,
				Message: name + " must be a valid uuid",
				Fields:  map[string][]string{name: {"must be a valid uuid"}},
			}
		}
	}
	return nil
}
