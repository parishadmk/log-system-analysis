package lib

import (
    "github.com/gocql/gocql"
)

func NewCassandraSession(hosts []string) (*gocql.Session, error) {
    cluster := gocql.NewCluster(hosts...)
    cluster.Consistency = gocql.LocalOne
    return cluster.CreateSession()
}