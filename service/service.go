package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo Repo
}

type Repo interface {
	CreateApplication(context.Context, Application) error
	GetApplication(context.Context, uuid.UUID) (*Application, error)
	ListApplication(context.Context, ApplicationFilter) ([]*Application, int, error)
	UpdateApplication(context.Context, Application) error

	ListApplicationTypes(ctx context.Context, filters ApplicationFilter) ([]ApplicationType, int, error)
	ListApplicationSubTypes(ctx context.Context, filters ApplicationFilter) ([]ApplicationSubType, int, error)

	CreateUser(ctx context.Context, user User) error
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	ListUser(ctx context.Context, filters UserFilter) ([]*User, int, error)
	DeleteUser(ctx context.Context, id uuid.UUID, currentTime time.Time) error
}

func (s *Service) SetTransaction(repo Repo) *Service {
	srv := &Service{}
	*srv = *s
	srv.repo = repo
	return srv
}

var ErrNotFound = errors.New("NotFound")

func NewService(repo Repo) *Service {
	return &Service{
		repo: repo,
	}
}
