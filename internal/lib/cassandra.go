package lib

import (
	"github.com/gocql/gocql"
)

// NewCassandraSession takes a slice of hostnames (no ports) and connects on 9042.
func NewCassandraSession(hosts []string) (*gocql.Session, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Port = 9042 // force the default C* port
	cluster.Consistency = gocql.LocalOne
	return cluster.CreateSession()
}
