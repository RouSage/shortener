package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (q *Queries) IsDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}

func (q *Queries) IsCheckConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23514"
	}

	return false
}

func (q *Queries) IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
