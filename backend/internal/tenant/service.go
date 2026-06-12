package tenant

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Service contains business logic for tenant operations.
type Service struct {
	repo *SqlcRepository
}

// NewService creates a new Service.
func NewService(repo *SqlcRepository) *Service {
	return &Service{repo: repo}
}

// CreateTenant creates a new tenant record.
func (s *Service) CreateTenant(ctx context.Context, name string, slug string) (*Tenant, error) {
	if slug == "" {
		slug = generateSlug(name)
	}

	// Check existence by name
	exists, err := s.repo.ExistsByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("check tenant exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("tenant with name %q already exists", name)
	}

	// Ensure slug is unique
	baseSlug := slug
	for i := 2; ; i++ {
		slugExists, err := s.repo.ExistsBySlug(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("check slug exists: %w", err)
		}
		if !slugExists {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	tenant, err := s.repo.Create(ctx, name, slug)
	if err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	return tenant, nil
}

// generateSlug creates a URL-friendly slug from a name.
func generateSlug(name string) string {
	var slug strings.Builder
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			slug.WriteRune(r + 32)
		} else if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			slug.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			slug.WriteRune('-')
		}
	}
	if slug.Len() == 0 {
		slug.WriteString(fmt.Sprintf("school-%d", time.Now().UnixNano()))
	}
	return slug.String()
}
