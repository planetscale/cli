package connections

import (
	"errors"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

// ValidateEngineFlags rejects flags that only apply to another database engine.
func ValidateEngineFlags(engine ps.DatabaseEngine, filter ConnectionFilter, target ConnectionTarget) error {
	return validateEngineFlags(engine, filter.connectionFilter(), target)
}

func validateEngineFlags(engine ps.DatabaseEngine, filter connectionFilter, target ConnectionTarget) error {
	switch engine {
	case ps.DatabaseEnginePostgres:
		if target.Keyspace != "" || target.Shard != "" {
			return errors.New("--keyspace/--shard are only supported for Vitess databases")
		}
	case ps.DatabaseEngineMySQL:
		if filter.active() {
			return errors.New("--instance/--role are only supported for Postgres databases")
		}
	}
	return nil
}
