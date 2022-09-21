package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rs/zerolog"
)

type QueryerContext interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type Repo struct {
	tx QueryerContext
}

func NewRepo(client QueryerContext) *Repo {
	return &Repo{
		tx: client,
	}
}

func (r *Repo) NewTransaction(ctx context.Context) (*Repo, error) {
	switch db := r.tx.(type) {
	case *sql.DB:
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		return NewRepo(tx), nil
	default:
		return NewRepo(db), nil
	}
}

func (s *Repo) Rollback(ctx context.Context) {
	tx, ok := s.tx.(interface{ Rollback() error })
	if !ok {
		return
	}

	err := tx.Rollback()
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("rollback is failed")
	}
}

func (s *Repo) Commit() error {
	tx, ok := s.tx.(interface{ Commit() error })
	if !ok {
		return nil
	}

	return tx.Commit()
}

func count(ctx context.Context, tx QueryerContext, query string, args []interface{}) (int, error) {
	total := sql.NullInt32{}

	err := tx.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}

	return int(total.Int32), nil
}
