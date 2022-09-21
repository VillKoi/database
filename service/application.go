package service

import (
	"bio/pagination"
	"context"
	"time"

	"github.com/google/uuid"
)

type ApplicationStatus string

const (
	ApplStatusCreated    ApplicationStatus = "created"
	ApplStatusInProgress ApplicationStatus = "inprogress"
	ApplStatusDone       ApplicationStatus = "done"
)

type Application struct {
	ID        uuid.UUID
	CreatedAt time.Time
	CreatorID uuid.UUID
	UpdatedAt time.Time

	Status  ApplicationStatus
	Type    uuid.UUID
	SubType uuid.UUID

	Text string

	// TODO: в отдельную таблицу вставлять фотографии
	PhotoIDs []uuid.UUID

	PerformerID   *uuid.UUID
	PerformerTime *time.Time
}

type ApplicationFilter struct {
	PerformerID *uuid.UUID
	CreatorID   *uuid.UUID
	Status      ApplicationStatus
	Type        *uuid.UUID

	Pagination pagination.Pagination
}

type ApplicationType struct {
	ID    uuid.UUID
	Title string
}

type ApplicationSubType struct {
	ID    uuid.UUID
	Title string
	Type  uuid.UUID
}

func (s *Service) CreateApplication(ctx context.Context, appl Application) (*Application, error) {
	err := s.repo.CreateApplication(ctx, appl)
	if err != nil {
		return nil, err
	}

	return s.repo.GetApplication(ctx, appl.ID)
}

func (s *Service) GetApplication(ctx context.Context, id uuid.UUID) (*Application, error) {
	return s.repo.GetApplication(ctx, id)
}

func (s *Service) UpdateApplication(ctx context.Context, appl Application) (*Application, error) {
	err := s.repo.UpdateApplication(ctx, appl)
	if err != nil {
		return nil, err
	}

	return s.repo.GetApplication(ctx, appl.ID)
}

func (s *Service) ListApplication(ctx context.Context, filter ApplicationFilter) ([]*Application, int, error) {
	return s.repo.ListApplication(ctx, filter)
}

func (s *Service) ListApplicationTypes(ctx context.Context, filter ApplicationFilter) ([]ApplicationType, int, error) {
	return s.repo.ListApplicationTypes(ctx, filter)
}

func (s *Service) ListApplicationSubtypes(ctx context.Context, filter ApplicationFilter) ([]ApplicationSubType, int, error) {
	return s.repo.ListApplicationSubTypes(ctx, filter)
}
