package services

import (
	"context"
)

type GuardianLinkService struct{}

func NewGuardianLinkService() *GuardianLinkService {
	return &GuardianLinkService{}
}

type GuardianLinkRequest struct {
	StudentID            string
	GuardianMembershipID string
	RelationshipType     string
}

func (s *GuardianLinkService) LinkGuardian(ctx context.Context, req GuardianLinkRequest) error {
	return nil
}
