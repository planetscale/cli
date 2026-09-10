package connections

import (
	"errors"

	ps "github.com/planetscale/cli/internal/planetscale"
)

var errNekiUnsupported = errors.New("connections is not supported for Neki databases")

// ValidateEngineFlags rejects flags that only apply to another database engine.
func ValidateEngineFlags(engine ps.DatabaseEngine, filter ConnectionFilter, target ConnectionTarget) error {
	return validateEngineFlags(engine, filter.connectionFilter(), target)
}

func validateEngineFlags(engine ps.DatabaseEngine, filter connectionFilter, target ConnectionTarget) error {
	if engine == ps.DatabaseEngineNeki {
		return errNekiUnsupported
	}
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
