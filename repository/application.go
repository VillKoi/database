package repository

import (
	"bio/service"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vagruchi/sqb"
)

func (r *Repo) CreateApplication(ctx context.Context, appl service.Application) error {
	query := `INSERT INTO application (id, created_at, creator_id, status, type, subtype, text)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.tx.ExecContext(ctx, query,
		appl.ID, appl.CreatedAt, appl.CreatorID, appl.Status, appl.Type, appl.SubType, appl.Text)

	return err
}

func (r *Repo) GetApplication(ctx context.Context, id uuid.UUID) (*service.Application, error) {
	query := `SELECT id, created_at, creator_id, updated_at, status, type, subtype, text, performer_id, performer_time
	FROM application AS a
	WHERE a.id = $1`

	rows, err := r.tx.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appl := &service.Application{}

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	err = rows.Scan(&appl.ID, &appl.CreatedAt, &appl.CreatorID, &appl.UpdatedAt, &appl.Status, &appl.Type, &appl.SubType, &appl.Text,
		&appl.PerformerID, &appl.PerformerTime)
	if err != nil {
		return nil, err
	}

	return appl, nil
}

func addApplicationFilters(q *sqb.SelectStmt, filters service.ApplicationFilter, isCount bool) *sqb.SelectStmt {
	query := *q

	if filters.PerformerID != nil {
		query = query.Where(append(query.WhereStmt.Exprs, sqb.Eq(sqb.Column("a.performer_id"), sqb.Arg{V: *filters.PerformerID}))...)
	}

	if filters.CreatorID != nil {
		query = query.Where(append(query.WhereStmt.Exprs, sqb.Eq(sqb.Column("a.creator_id"), sqb.Arg{V: *filters.CreatorID}))...)
	}

	if filters.Status != "" {
		query = query.Where(append(query.WhereStmt.Exprs, sqb.Eq(sqb.Column(`a.status`), sqb.Arg{V: filters.Status}))...)
	}

	if filters.Type != nil {
		query = query.Where(append(query.WhereStmt.Exprs, sqb.Eq(sqb.Column("a.type"), sqb.Arg{V: *filters.Type}))...)
	}

	if !isCount {
		if len(filters.Pagination.OrderBy) == 0 {
			filters.Pagination.AddOrderByAsc(`a.created_at`)
		}
		query = *filters.Pagination.Apply(&query)
	}

	return &query
}

func (r *Repo) countApplications(ctx context.Context, filters service.ApplicationFilter) (int, error) {
	query := sqb.From(
		sqb.JB(sqb.TableName(`application`).As(`a`)).
			InnerJoin(sqb.TableName(`application_type`).As(`at`), sqb.Eq(sqb.Column(`a.type`), sqb.Column(`at.id`)))).
		Select(sqb.Count(sqb.Column(`a.id`)))

	query = *addApplicationFilters(&query, filters, true)

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return 0, err
	}

	return count(ctx, r.tx, rawquery, args)
}

func (r *Repo) ListApplication(ctx context.Context, filters service.ApplicationFilter) ([]*service.Application, int, error) {
	total, err := r.countApplications(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	query := sqb.From(
		sqb.JB(sqb.TableName(`application`).As(`a`)).
			InnerJoin(sqb.TableName(`application_type`).As(`at`), sqb.Eq(sqb.Column(`a.type`), sqb.Column(`at.id`)))).
		Select(sqb.Column(`a.id`), sqb.Column(`a.created_at`), sqb.Column(`a.creator_id`), sqb.Column(`a.updated_at`),
			sqb.Column(`a.status`), sqb.Column(`a.type`), sqb.Column(`a.subtype`), sqb.Column(`a.text`),
			sqb.Column(`a.performer_id`), sqb.Column(`a.performer_time`))

	query = *addApplicationFilters(&query, filters, false)

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.tx.QueryContext(ctx, rawquery, args...)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	applications := []*service.Application{}

	for rows.Next() {
		appl := &service.Application{}

		err = rows.Scan(&appl.ID, &appl.CreatedAt, &appl.CreatorID, &appl.UpdatedAt,
			&appl.Status, &appl.Type, &appl.SubType, &appl.Text, &appl.PerformerID, &appl.PerformerTime)
		if err != nil {
			return nil, 0, err
		}
		applications = append(applications, appl)
	}

	err = rows.Err()
	if err != nil {
		return nil, 0, err
	}

	return applications, total, nil
}

func (r *Repo) UpdateApplication(ctx context.Context, appl service.Application) error {
	uptTime := time.Now().UTC()

	update := sqb.UpdateStmt{
		Table: sqb.TableName("application"),
		Set: sqb.SetStmt{
			{
				Key:   sqb.Column("updated_at"),
				Value: sqb.Arg{V: uptTime},
			},
		},
		WhereStmt: sqb.WhereStmt{
			Exprs: []sqb.BoolExpr{sqb.Eq(
				sqb.Column("id"), sqb.Arg{V: appl.ID},
			)},
		},
	}

	if appl.Status != "" {
		update.Set = append(update.Set, sqb.SetArg{
			Key:   sqb.Column(`status`),
			Value: sqb.Arg{V: appl.Status},
		})
	}

	if appl.PerformerID != nil {
		update.Set = append(update.Set, sqb.SetArg{
			Key:   sqb.Column(`performer_id`),
			Value: sqb.Arg{V: *appl.PerformerID},
		})
	}

	if appl.PerformerTime != nil {
		update.Set = append(update.Set, sqb.SetArg{
			Key:   sqb.Column(`performer_time`),
			Value: sqb.Arg{V: appl.PerformerTime},
		})
	}

	if len(update.Set) == 1 {
		return errors.New("nothing update")
	}

	rawQuery, args, err := sqb.ToPostgreSql(update)
	if err != nil {
		return err
	}

	_, err = r.tx.ExecContext(ctx, rawQuery, args...)

	return err
}

func (r *Repo) countApplicationTypes(ctx context.Context, filters service.ApplicationFilter) (int, error) {
	query := sqb.From(sqb.TableName(`application_type`).As(`at`)).
		Select(sqb.Count(sqb.Column(`at.id`)))

	query = *addApplicationFilters(&query, filters, true)

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return 0, err
	}

	fmt.Println(rawquery)

	return count(ctx, r.tx, rawquery, args)
}

func (r *Repo) ListApplicationTypes(ctx context.Context, filters service.ApplicationFilter) ([]service.ApplicationType, int, error) {
	total, err := r.countApplicationTypes(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	query := sqb.From(sqb.TableName(`application_type`).As(`at`)).
		Select(sqb.Column(`at.id`), sqb.Column(`at.title`))

	query = *addApplicationFilters(&query, filters, false)

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return nil, 0, err
	}

	fmt.Println(rawquery)

	rows, err := r.tx.QueryContext(ctx, rawquery, args...)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	applicationTypes := []service.ApplicationType{}

	for rows.Next() {
		addr := service.ApplicationType{}

		err = rows.Scan(&addr.ID, &addr.Title)
		if err != nil {
			return nil, 0, err
		}
		applicationTypes = append(applicationTypes, addr)
	}

	err = rows.Err()
	if err != nil {
		return nil, 0, err
	}

	return applicationTypes, total, nil
}

func (r *Repo) countApplicationSubTypes(ctx context.Context, filters service.ApplicationFilter) (int, error) {
	query := sqb.From(
		sqb.JB(sqb.TableName(`application_subtype`).As(`ast`)).
			InnerJoin(sqb.TableName(`application_type`).As(`at`), sqb.Eq(sqb.Column(`ast.type`), sqb.Column(`at.id`)))).
		Select(sqb.Count(sqb.Column(`ast.id`)))

	query = *addApplicationFilters(&query, filters, true)

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return 0, err
	}

	fmt.Println(rawquery)

	return count(ctx, r.tx, rawquery, args)
}

func (r *Repo) ListApplicationSubTypes(ctx context.Context, filters service.ApplicationFilter) ([]service.ApplicationSubType, int, error) {
	total, err := r.countApplicationSubTypes(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	query := sqb.From(
		sqb.JB(sqb.TableName(`application_subtype`).As(`ast`)).
			InnerJoin(sqb.TableName(`application_type`).As(`at`), sqb.Eq(sqb.Column(`ast.type`), sqb.Column(`at.id`)))).
		Select(sqb.Column(`ast.id`), sqb.Column(`ast.title`), sqb.Column(`ast.type`))

	query = *addApplicationFilters(&query, filters, false)

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return nil, 0, err
	}

	fmt.Println(rawquery)

	rows, err := r.tx.QueryContext(ctx, rawquery, args...)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	applicationsSubtypes := []service.ApplicationSubType{}

	for rows.Next() {
		addr := service.ApplicationSubType{}

		err = rows.Scan(&addr.ID, &addr.Title, &addr.Type)
		if err != nil {
			return nil, 0, err
		}
		applicationsSubtypes = append(applicationsSubtypes, addr)
	}

	err = rows.Err()
	if err != nil {
		return nil, 0, err
	}

	return applicationsSubtypes, total, nil
}
