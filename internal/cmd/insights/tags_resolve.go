package insights

import (
	"fmt"
	"strings"

	ps "github.com/planetscale/cli/internal/planetscale"
)

// tagSourcePrefixes maps UI/source names to the API dimension id prefix.
var tagSourcePrefixes = map[string]string{
	"sql":    "S",
	"system": "B",
	"sys":    "B",
}

// resolveTagIDs maps friendly tag names (as shown in the app Key picker) to
// API dimension ids (e.g. "app" -> "Sapp"). Inputs may also be:
//   - already-prefixed ids ("Sapp", "Busername")
//   - source-qualified names ("sql:app", "system:username")
func resolveTagIDs(available []*ps.QueryTag, inputs []string) ([]string, error) {
	byID := make(map[string]*ps.QueryTag, len(available))
	byName := make(map[string][]*ps.QueryTag, len(available))
	for _, t := range available {
		byID[t.ID] = t
		byName[t.Name] = append(byName[t.Name], t)
	}

	ids := make([]string, 0, len(inputs))
	for _, raw := range inputs {
		input := strings.TrimSpace(raw)
		if input == "" {
			continue
		}

		if t, ok := byID[input]; ok {
			ids = append(ids, t.ID)
			continue
		}

		source, name, hasSource := strings.Cut(input, ":")
		if hasSource {
			prefix, ok := tagSourcePrefixes[strings.ToLower(source)]
			if !ok {
				return nil, fmt.Errorf("invalid tag source %q in %q; use sql: or system: (e.g. sql:app)", source, input)
			}
			id := prefix + name
			if _, ok := byID[id]; !ok {
				return nil, fmt.Errorf("tag %q not found; run 'pscale insights tags' to list available keys", input)
			}
			ids = append(ids, id)
			continue
		}

		// Prefer friendly names (UI Key picker) before treating S*/B* as ids,
		// so names like "Status" still resolve via byName.
		matches := byName[input]
		switch len(matches) {
		case 1:
			ids = append(ids, matches[0].ID)
			continue
		case 0:
			// Fall through to prefixed-id passthrough for agents.
		default:
			opts := make([]string, 0, len(matches))
			for _, m := range matches {
				opts = append(opts, m.Source+":"+m.Name)
			}
			return nil, fmt.Errorf("tag %q matches multiple sources (%s); disambiguate with sql:%s or system:%s",
				input, strings.Join(opts, ", "), input, input)
		}

		// Bare prefixed id not in the current list still accepted so agents
		// can pass ids from a prior list call.
		if len(input) > 1 && (input[0] == 'S' || input[0] == 'B') {
			ids = append(ids, input)
			continue
		}

		return nil, fmt.Errorf("tag %q not found; run 'pscale insights tags' to list available keys", input)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one --tags value is required")
	}
	return ids, nil
}
