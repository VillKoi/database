package respond

import (
	"bio/specs"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.services.mts.ru/abp/myosotis/logger"
)

var (
	NoUserInTokenErr = fmt.Errorf("no user in token")
)

func withJSON(ctx context.Context, w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		err := json.NewEncoder(w).Encode(payload)
		if err != nil {
			logger.GetLogger(ctx).WithError(err).Error("write answer")
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
