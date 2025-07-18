package lib

import (
	"context"
    "github.com/jackc/pgx/v5/pgxpool"
)

func NewCockroachPool(dsn string) (*pgxpool.Pool, error) {
    return pgxpool.New(context.Background(), dsn)
}