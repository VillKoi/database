package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserRole string

type User struct {
	ID        uuid.UUID
	CreatedAt time.Time
	FirstName string
	LastName  string
	Role      UserRole
	Phone     string
}

type UserFilter struct {
	ID *uuid.UUID
}

func (s *Service) CreateUser(ctx context.Context, user User) (*User, error) {
	err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return s.repo.GetUser(ctx, user.ID)
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetUser(ctx, id)
}

func (s *Service) ListUsers(ctx context.Context, filter UserFilter) ([]*User, int, error) {
	return s.repo.ListUser(ctx, filter)
}
