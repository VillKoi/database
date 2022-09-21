package repository

import (
	"bio/service"
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/vagruchi/sqb"
)

func (r *Repo) CreateUser(ctx context.Context, user service.User) error {
	query := `INSERT INTO users (id, created_at, first_name, last_name, role, phone)
	VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.tx.ExecContext(ctx, query,
		user.ID, user.CreatedAt, user.FirstName, user.LastName, user.Role, user.Phone)

	return err
}

func (r *Repo) GetUser(ctx context.Context, id uuid.UUID) (*service.User, error) {
	query := `SELECT id, created_at, first_name, last_name, role, phone
	FROM users AS u
	WHERE u.id = $1`

	rows, err := r.tx.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user := &service.User{}

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	err = rows.Scan(&user.ID, &user.CreatedAt, &user.FirstName, &user.LastName, &user.Role, &user.Phone)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func addUserFilters(q *sqb.SelectStmt, filters service.UserFilter, isCount bool) *sqb.SelectStmt {
	query := *q

	query = query.Where(append(query.WhereStmt.Exprs, sqb.Raw(`u.deleted_at IS NULL`))...)

	if filters.ID != nil {
		query = query.Where(append(query.WhereStmt.Exprs, sqb.Eq(sqb.Column(`u.id`), sqb.Arg{V: *filters.ID}))...)
	}

	return &query
}

func (r *Repo) countUsers(ctx context.Context, filters service.UserFilter) (int, error) {
	query := sqb.From(sqb.TableName(`users`).As(`u`)).
		Select(sqb.Count(sqb.Column(`u.id`)))

	query = *addUserFilters(&query, filters, false)

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return 0, err
	}

	return count(ctx, r.tx, rawquery, args)
}

func (r *Repo) ListUser(ctx context.Context, filters service.UserFilter) ([]*service.User, int, error) {
	total, err := r.countUsers(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	query := sqb.From(sqb.TableName(`users`).As(`u`)).
		Select(sqb.Column(`u.id`), sqb.Column(`u.created_at`), sqb.Column(`u.first_name`), sqb.Column(`u.last_name`),
			sqb.Column(`u.role`), sqb.Column(`u.phone`))

	query = *addUserFilters(&query, filters, false)

	q, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}

	users := []*service.User{}

	for rows.Next() {
		user := &service.User{}

		err := rows.Scan(&user.ID, &user.CreatedAt, &user.FirstName, &user.LastName, &user.Role, &user.Phone)
		if err != nil {
			return nil, 0, err
		}

		users = append(users, user)
	}

	return users, total, nil
}

func (r *Repo) DeleteUser(ctx context.Context, id uuid.UUID, currentTime time.Time) error {
	query := `UPDATE users
	SET deleted_at = $1
	WHERE id = $2`

	_, err := r.tx.ExecContext(ctx, query,
		id, currentTime)

	return err
}
