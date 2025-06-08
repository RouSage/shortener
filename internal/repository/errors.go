package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func (q *Queries) IsDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
