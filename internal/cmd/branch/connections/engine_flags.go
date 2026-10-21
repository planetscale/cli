package connections

import (
	"errors"

	ps "github.com/planetscale/cli/internal/planetscale"
)

var (
	errVitessTargetOnly   = errors.New("--keyspace/--shard are only supported for Vitess databases")
	errPostgresFilterOnly = errors.New("--instance/--role are only supported for Postgres databases")
)

// ValidateEngineFlags rejects flags that only apply to another database engine.
func ValidateEngineFlags(engine ps.DatabaseEngine, filter ConnectionFilter, target ConnectionTarget) error {
	return validateEngineFlags(engine, filter.connectionFilter(), target)
}

func validateEngineFlags(engine ps.DatabaseEngine, filter connectionFilter, target ConnectionTarget) error {
	vitessTarget := target.Keyspace != "" || target.Shard != ""
	switch engine {
	case ps.DatabaseEnginePostgres:
		if vitessTarget {
			return errVitessTargetOnly
		}
	case ps.DatabaseEngineMySQL:
		if filter.active() {
			return errPostgresFilterOnly
		}
	case ps.DatabaseEngineNeki:
		if vitessTarget {
			return errVitessTargetOnly
		}
		if filter.active() {
			return errPostgresFilterOnly
		}
	}
	return nil
}
