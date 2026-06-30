package students

import (
	"context"
	"fmt"
)

// Service implements student business logic.
type Service struct {
	repo StudentRepository
}

// NewService creates a new Service.
func NewService(repo StudentRepository) *Service {
	return &Service{repo: repo}
}

// ListStudents returns a paginated list of students.
func (s *Service) ListStudents(ctx context.Context, filter ListFilter) (ListStudentsResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 50
	}

	students, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return ListStudentsResponse{}, fmt.Errorf("students.Service.ListStudents: %w", err)
	}

	return ListStudentsResponse{
		Students: students,
		Total:    total,
		Page:     filter.Page,
		Limit:    filter.Limit,
	}, nil
}
