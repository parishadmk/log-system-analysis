package lib

import (
    "database/sql"
    _ "github.com/ClickHouse/clickhouse-go"
)

func NewClickHouseConn(dsn string) (*sql.DB, error) {
    return sql.Open("clickhouse", dsn)
}