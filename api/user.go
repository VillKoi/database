package api

import (
	"bio/service"
	"bio/specs"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gitlab.services.mts.ru/libs/logger"
)

func (ctrl *Controller) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := ApiToUser(ctx, r.Body)
	if err != nil {
		WithBadRequestError(ctx, w, err.Error())
		return
	}

	srvc, repo, err := ctrl.createTxService(ctx)
	if err != nil {
		fmt.Println("create tx: ", err)
		WithInternalServerError(ctx, w, "")
		return
	}

	createdUser, err := srvc.CreateUser(ctx, *user)
	switch err {
	case nil:
		err = repo.Commit()
		if err != nil {
			fmt.Println("cannot commit result: ", err)
			WithInternalServerError(ctx, w, http.StatusText(http.StatusInternalServerError))
			return
		}
		res := UserToApi(createdUser)
		WithStatusOK(ctx, w, res)
	default:
		repo.Rollback(ctx)
		fmt.Println("create user: ", err)
		WithInternalServerError(ctx, w, "")
	}
	return
}

func ApiToUser(ctx context.Context, body io.ReadCloser) (*service.User, error) {
	entry := logger.GetLogger(ctx)
	reqUser := specs.CreateUserPayload{}

	err := json.NewDecoder(body).Decode(&reqUser)
	if err != nil {
		entry.WithError(err).Warning("get article json body")
		return nil, errors.New("incorrect json")
	}

	if reqUser.FirstName == "" {
		entry.Warning("empty Text")
		return nil, errors.New("empty Text")
	}

	if reqUser.LastName == "" {
		entry.Warning("empty description")
		return nil, errors.New("empty description")
	}

	appl := &service.User{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),

		FirstName: reqUser.FirstName,
		LastName:  reqUser.LastName,
		Phone:     reqUser.Phone,
		// TODO:
		Role: service.UserRole(reqUser.Role),
	}

	return appl, nil
}

func UserToApi(in *service.User) specs.UserResponse {
	return specs.UserResponse{
		Id:        in.ID.String(),
		CreatedAt: in.CreatedAt,
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Phone:     in.Phone,
		Role:      specs.UserRole(in.Role),
	}
}

func (ctrl *Controller) GetUser(w http.ResponseWriter, r *http.Request, userId string) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	id, err := uuid.Parse(userId)
	if err != nil {
		logger.Warn().Err(err).Msg("parse application id")
		WithBadRequestError(ctx, w, "invalid application id")
		return
	}

	srvc, repo, err := ctrl.createTxService(ctx)
	if err != nil {
		fmt.Println("create tx: ", err)
		WithInternalServerError(ctx, w, "")
		return
	}

	createdUser, err := srvc.GetUser(ctx, id)
	switch err {
	case nil:
		err = repo.Commit()
		if err != nil {
			fmt.Println("cannot commit result: ", err)
			WithInternalServerError(ctx, w, http.StatusText(http.StatusInternalServerError))
			return
		}
		res := UserToApi(createdUser)
		WithStatusOK(ctx, w, res)
	default:
		repo.Rollback(ctx)
		fmt.Println("get user: ", err)
		WithInternalServerError(ctx, w, "")
	}
	return
}

func (ctrl *Controller) ListUsers(w http.ResponseWriter, r *http.Request, params specs.ListUsersParams) {
	ctx := r.Context()
	// logger := zerolog.Ctx(ctx)

	filter := service.UserFilter{}

	srvc, repo, err := ctrl.createTxService(ctx)
	if err != nil {
		fmt.Println("create tx: ", err)
		WithInternalServerError(ctx, w, "")
		return
	}

	users, total, err := srvc.ListUsers(ctx, filter)
	switch err {
	case nil:
		err = repo.Commit()
		if err != nil {
			fmt.Println("cannot commit result: ", err)
			WithInternalServerError(ctx, w, http.StatusText(http.StatusInternalServerError))
			return
		}

		res := specs.ListUsersResponse{
			Data: arrayInArray(users, UserToApi),
			Meta: specs.ResponseMetaTotal{
				Total: total,
			},
		}
		WithStatusOK(ctx, w, res)
	default:
		repo.Rollback(ctx)
		fmt.Println("list user: ", err)
		WithInternalServerError(ctx, w, "")
	}
	return
}

func (ctrl *Controller) DeleteUser(w http.ResponseWriter, r *http.Request, userId string) {
}
