package service

import "github.com/jackc/pgx"

func isUniqueViolation(err error) bool {
	pgerror, ok := err.(pgx.PgError)
	return ok && pgerror.Code == "23505"
}
