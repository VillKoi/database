package api

import (
	"bio/auth"
	"bio/respond"
	"bio/service"
	"bio/specs"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"bio/pagination"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func GetApplicationPaginationPolitics() pagination.PaginationPolitics {
	return pagination.PaginationPolitics{
		MaxLimit:     50,
		DefaultLimit: 25,
		OrderByMappgin: map[string]string{
			"date_created": "date_created",
			"status":       "status",
		},
	}
}

func (ctrl *Controller) CreateApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	user, ok := auth.UserFromContext(ctx)
	if !ok {
		logger.Warn().Err(NoUserInTokenErr).Msg("get user from context")
		WithUnauthorizedError(ctx, w)
		return
	}

	createdApplication, err := ApiToCreationApplication(ctx, r.Body)
	if err != nil {
		WithBadRequestError(ctx, w, err.Error())
		return
	}

	createdApplication.CreatorID = user.ID

	srvc, repo, err := ctrl.createTxService(ctx)
	if err != nil {
		fmt.Println("create tx: ", err)
		WithInternalServerError(ctx, w, "")
		return
	}

	application, err := srvc.CreateApplication(ctx, *createdApplication)
	switch err {
	case nil:
		err = repo.Commit()
		if err != nil {
			fmt.Println("cannot commit result: ", err)
			WithInternalServerError(ctx, w, http.StatusText(http.StatusInternalServerError))
			return
		}
		res := ApplicationToAPI(application)
		WithStatusOK(ctx, w, res)
	default:
		repo.Rollback(ctx)
		fmt.Println("create application: ", err)
		WithInternalServerError(ctx, w, "")
	}
	return
}

func ApiToCreationApplication(ctx context.Context, body io.ReadCloser) (*service.Application, error) {
	entry := zerolog.Ctx(ctx)
	reqAppl := specs.CreateApplicationPayload{}

	err := json.NewDecoder(body).Decode(&reqAppl)
	if err != nil {
		entry.Warn().Err(err).Msg("get article json body")
		return nil, errors.New("incorrect json")
	}

	if reqAppl.Text == "" {
		entry.Warn().Msg("empty Text")
		return nil, errors.New("empty Text")
	}

	if reqAppl.Type == "" {
		entry.Warn().Msg("empty type")
		return nil, errors.New("empty type")
	}

	appl := &service.Application{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),

		Status: service.ApplStatusCreated,
		Text:   reqAppl.Text,
	}

	applType, err := uuid.Parse(reqAppl.Type)
	if err != nil {
		entry.Warn().Msg("empty type")
		return nil, errors.New("empty type")
	}

	appl.Type = applType

	applSubType, err := uuid.Parse(reqAppl.Subtype)
	if err != nil {
		entry.Warn().Msg("empty subtype")
		return nil, errors.New("empty subtype")
	}

	appl.SubType = applSubType

	if reqAppl.PhotoIds != nil {
		photoIDs, err := arrayInArrayWithError(*reqAppl.PhotoIds, uuid.Parse)
		if err != nil {
			return nil, errors.New("parse photo")
		}

		appl.PhotoIDs = photoIDs
	}

	return appl, nil
}

func ApplicationToAPI(in *service.Application) specs.ApplicationResponse {
	out := specs.ApplicationResponse{
		Id:        in.ID.String(),
		CreatedAt: in.CreatedAt,
		CreatorId: in.CreatorID.String(),
		UpdatedAt: in.UpdatedAt,

		Status:  StatusToApi(in.Status),
		Type:    in.Type.String(),
		Subtype: in.SubType.String(),
		Text:    in.Text,

		PerformerAt: in.PerformerTime,

		PhotoIds: arrayInArray(in.PhotoIDs, func(v uuid.UUID) string { return v.String() }),
	}

	if in.PerformerID != nil {
		out.PerformerId = toPoint(in.PerformerID.String())
	}

	return out
}

func toPoint[T any](t T) *T {
	return &t
}

func StatusToApi(in service.ApplicationStatus) specs.ApplicationStatus {
	return map[service.ApplicationStatus]specs.ApplicationStatus{
		service.ApplStatusCreated:    specs.ApplicationStatusCreated,
		service.ApplStatusInProgress: specs.ApplicationStatusInProgress,
		service.ApplStatusDone:       specs.ApplicationStatusDone,
	}[in]
}

func ApiToStatus(in specs.ApplicationStatus) service.ApplicationStatus {
	return map[specs.ApplicationStatus]service.ApplicationStatus{
		specs.ApplicationStatusCreated:    service.ApplStatusCreated,
		specs.ApplicationStatusDone:       service.ApplStatusDone,
		specs.ApplicationStatusInProgress: service.ApplStatusInProgress,
	}[in]
}

func (ctrl *Controller) GetApplication(w http.ResponseWriter, r *http.Request, applicationId string) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	id, err := uuid.Parse(applicationId)
	if err != nil {
		logger.Warn().Err(err).Msg("parse application id")
		WithBadRequestError(ctx, w, "invalid application id")
		return
	}

	article, err := ctrl.srvc.GetApplication(ctx, id)
	switch err {
	case service.ErrNotFound:
		WithNotFoundError(ctx, w, "application not found")
	case nil:
		res := ApplicationToAPI(article)
		WithStatusOK(ctx, w, res)
	default:
		logger.Error().Err(err).Msg("get application")
		WithInternalServerError(ctx, w, "")
	}
	return

}

func (ctrl *Controller) ListApplications(w http.ResponseWriter, r *http.Request, params specs.ListApplicationsParams) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	filter := service.ApplicationFilter{}

	if params.PerformerId != nil {
		performerId, err := uuid.Parse(*params.PerformerId)
		if err != nil {
			logger.Warn().Err(err).Msg("parse PerformerId")
			WithBadRequestError(ctx, w, "invalid PerformerId")
			return
		}

		filter.PerformerID = &performerId
	}

	if params.CreatorId != nil {
		creatorId, err := uuid.Parse(*params.CreatorId)
		if err != nil {
			logger.Warn().Err(err).Msg("parse CreatorId")
			WithBadRequestError(ctx, w, "invalid CreatorId")
			return
		}

		filter.CreatorID = &creatorId
	}

	if params.Status != nil {
		status := ApiToStatus(*params.Status)
		if status == "" {
			logger.Warn().Msg("empty status")
			WithBadRequestError(ctx, w, "invalid status")
			return
		}

		filter.Status = status
	}

	if params.Type != nil {
		typeId, err := uuid.Parse(*params.Type)
		if err != nil {
			logger.Warn().Err(err).Msg("parse Type")
			WithBadRequestError(ctx, w, "invalid Type")
			return
		}

		filter.Type = &typeId
	}

	pgnPolitics, err := GetApplicationPaginationPolitics().MakePagination(params.Pagination, params.Sort)
	if err != nil {
		respond.WithBadRequestError(ctx, w, err.Error())
		return
	}

	filter.Pagination = pgnPolitics

	srvc, repo, err := ctrl.createTxService(ctx)
	if err != nil {
		fmt.Println("create tx: ", err)
		WithInternalServerError(ctx, w, "")
		return
	}

	application, total, err := srvc.ListApplication(ctx, filter)
	switch err {
	case nil:
		err = repo.Commit()
		if err != nil {
			fmt.Println("cannot commit result: ", err)
			WithInternalServerError(ctx, w, http.StatusText(http.StatusInternalServerError))
			return
		}

		res := specs.ListApplicationResponse{
			Data: arrayInArray(application, ApplicationToAPI),
			Meta: specs.ResponseMetaTotal{
				Total: total,
			},
		}
		WithStatusOK(ctx, w, res)
	default:
		repo.Rollback(ctx)
		fmt.Println("list applications: ", err)
		WithInternalServerError(ctx, w, "")
	}
	return
}

func (ctrl *Controller) UpdateApplication(w http.ResponseWriter, r *http.Request, applicationId string) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	id, err := uuid.Parse(applicationId)
	if err != nil {
		logger.Warn().Err(err).Msg("parse application id")
		WithBadRequestError(ctx, w, "invalid application id")
		return
	}

	updatedApplication, err := ApiToUpdateApplication(ctx, r.Body)
	if err != nil {
		WithBadRequestError(ctx, w, err.Error())
		return
	}

	updatedApplication.ID = id

	srvc, repo, err := ctrl.createTxService(ctx)
	if err != nil {
		fmt.Println("create tx: ", err)
		WithInternalServerError(ctx, w, "")
		return
	}

	application, err := srvc.UpdateApplication(ctx, *updatedApplication)
	switch err {
	case nil:
		err = repo.Commit()
		if err != nil {
			fmt.Println("cannot commit result: ", err)
			WithInternalServerError(ctx, w, http.StatusText(http.StatusInternalServerError))
			return
		}
		res := ApplicationToAPI(application)
		WithStatusOK(ctx, w, res)
	default:
		repo.Rollback(ctx)
		fmt.Println("update application: ", err)
		WithInternalServerError(ctx, w, "")
	}
	return
}

func ApiToUpdateApplication(ctx context.Context, body io.ReadCloser) (*service.Application, error) {
	entry := zerolog.Ctx(ctx)
	reqAppl := specs.UpdateApplicationPayload{}

	err := json.NewDecoder(body).Decode(&reqAppl)
	if err != nil {
		entry.Warn().Err(err).Msg("get article json body")
		return nil, errors.New("incorrect json")
	}

	appl := &service.Application{
		UpdatedAt:     time.Now().UTC(),
		PerformerTime: reqAppl.PerformerTime,
	}

	if reqAppl.PerformerId != nil {
		performerID, err := uuid.Parse(*reqAppl.PerformerId)
		if err != nil {
			return nil, errors.New("parse performer id")
		}

		appl.PerformerID = &performerID
	}

	if reqAppl.Status != nil {
		status := ApiToStatus(*reqAppl.Status)

		appl.Status = status
	}

	return appl, nil
}

func (ctrl *Controller) ListApplicationTypes(w http.ResponseWriter, r *http.Request, params specs.ListApplicationTypesParams) {
	ctx := r.Context()
	// logger := zerolog.Ctx(ctx)

	filter := service.ApplicationFilter{}

	srvc, repo, err := ctrl.createTxService(ctx)
	if err != nil {
		fmt.Println("create tx: ", err)
		WithInternalServerError(ctx, w, "")
		return
	}

	applicationTypes, total, err := srvc.ListApplicationTypes(ctx, filter)
	switch err {
	case nil:
		err = repo.Commit()
		if err != nil {
			fmt.Println("cannot commit result: ", err)
			WithInternalServerError(ctx, w, http.StatusText(http.StatusInternalServerError))
			return
		}

		res := specs.ListApplicationTypes{
			Data: arrayInArray(applicationTypes, ApplicationTypeToAPI),
			Meta: specs.ResponseMetaTotal{
				Total: total,
			},
		}
		WithStatusOK(ctx, w, res)
	default:
		repo.Rollback(ctx)
		fmt.Println("list applications: ", err)
		WithInternalServerError(ctx, w, "")
	}
	return
}

func ApplicationTypeToAPI(in service.ApplicationType) specs.ApplicationType {
	return specs.ApplicationType{
		Id:    in.ID.String(),
		Title: in.Title,
	}
}

func ApplicationSubTypeToAPI(in service.ApplicationSubType) specs.ApplicationSubtype {
	return specs.ApplicationSubtype{
		Id:    in.ID.String(),
		Title: in.Title,
		Type:  in.Type.String(),
	}
}

func (ctrl *Controller) ListApplicationSubTypes(w http.ResponseWriter, r *http.Request, params specs.ListApplicationSubTypesParams) {
	ctx := r.Context()
	// logger := zerolog.Ctx(ctx)

	filter := service.ApplicationFilter{}

	srvc, repo, err := ctrl.createTxService(ctx)
	if err != nil {
		fmt.Println("create tx: ", err)
		WithInternalServerError(ctx, w, "")
		return
	}

	application, total, err := srvc.ListApplicationSubtypes(ctx, filter)
	switch err {
	case nil:
		err = repo.Commit()
		if err != nil {
			fmt.Println("cannot commit result: ", err)
			WithInternalServerError(ctx, w, http.StatusText(http.StatusInternalServerError))
			return
		}

		res := specs.ListApplicationSubtypes{
			Data: arrayInArray(application, ApplicationSubTypeToAPI),
			Meta: specs.ResponseMetaTotal{
				Total: total,
			},
		}
		WithStatusOK(ctx, w, res)
	default:
		repo.Rollback(ctx)
		fmt.Println("list applications: ", err)
		WithInternalServerError(ctx, w, "")
	}
	return
}
