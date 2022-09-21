package api

import (
	"bio/service"
	"bio/specs"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"bio/repository"

	"github.com/rs/zerolog"
)

var NoUserInTokenErr = fmt.Errorf("no user in token")

type Controller struct {
	_ specs.ServerInterface

	srvc *service.Service
	repo *repository.Repo
}

func NewController(stvc *service.Service, repo *repository.Repo) *Controller {
	return &Controller{
		srvc: stvc,
		repo: repo,
	}
}

var _ specs.ServerInterface = &Controller{}

func (c *Controller) createTxService(ctx context.Context) (*service.Service, *repository.Repo, error) {
	repTx, err := c.repo.NewTransaction(ctx)
	if err != nil {
		return nil, nil, err
	}

	srvc := c.srvc.SetTransaction(repTx)
	return srvc, repTx, nil
}

func arrayInArrayWithError[T any, R any](array []T, transform func(T) (R, error)) ([]R, error) {
	if len(array) == 0 {
		return nil, nil
	}

	arr2 := make([]R, len(array))
	var err error

	for i := range array {
		arr2[i], err = transform(array[i])
		if err != nil {
			return nil, err
		}
	}

	return arr2, nil
}

func arrayInArray[T any, R any](array []T, transform func(T) R) []R {
	if len(array) == 0 {
		return []R{}
	}

	arr2 := make([]R, len(array))

	for i := range array {
		arr2[i] = transform(array[i])
	}

	return arr2
}

func withJSON(ctx context.Context, w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		err := json.NewEncoder(w).Encode(payload)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("write answer")
		}
	}
}

func WithStatusOK(ctx context.Context, w http.ResponseWriter, payload interface{}) {
	withJSON(ctx, w, http.StatusOK, payload)
}

func WithError(ctx context.Context, w http.ResponseWriter, code int, message string) {
	resp := specs.Error{
		Code:    code,
		Message: message,
	}

	withJSON(ctx, w, code, resp)
}

func WithInternalServerError(ctx context.Context, w http.ResponseWriter, message string) {
	WithError(ctx, w, http.StatusInternalServerError, message)
}

func WithUnauthorizedError(ctx context.Context, w http.ResponseWriter) {
	WithError(ctx, w, http.StatusUnauthorized, "unauthorized")
}

func WithStatusConflictError(ctx context.Context, w http.ResponseWriter, message string) {
	WithError(ctx, w, http.StatusConflict, message)
}

func WithNotFoundError(ctx context.Context, w http.ResponseWriter, message string) {
	WithError(ctx, w, http.StatusNotFound, message)
}

func WithBadRequestError(ctx context.Context, w http.ResponseWriter, message string) {
	WithError(ctx, w, http.StatusBadRequest, message)
}
