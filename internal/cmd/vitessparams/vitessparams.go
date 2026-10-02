// Package vitessparams holds the parsing and output shared by the commands
// that change Vitess parameters on keyspaces and VTGates.
package vitessparams

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	ps "github.com/planetscale/cli/internal/planetscale"
)

// ParseChanges groups --parameters and --reset values by namespace. A nil
// value resets the parameter to its default. example is a valid
// --parameters value shown in error messages.
func ParseChanges(sets, resets, namespaces []string, example string) (map[string]map[string]*string, error) {
	if len(sets) == 0 && len(resets) == 0 {
		return nil, fmt.Errorf("pass at least one --parameters namespace.name=value or --reset namespace.name")
	}

	changes := make(map[string]map[string]*string)
	add := func(flag, raw, key string, value *string) error {
		namespace, name, found := strings.Cut(key, ".")
		if !found || namespace == "" || name == "" {
			return fmt.Errorf("invalid %s %q: parameter must be prefixed with its namespace, e.g. %s.%s", flag, raw, namespaces[0], key)
		}
		if !slices.Contains(namespaces, namespace) {
			return fmt.Errorf("invalid %s %q: namespace must be one of: %s", flag, raw, strings.Join(namespaces, ", "))
		}
		if _, exists := changes[namespace][name]; exists {
			return fmt.Errorf("parameter %s.%s is passed more than once", namespace, name)
		}
		if changes[namespace] == nil {
			changes[namespace] = make(map[string]*string)
		}
		changes[namespace][name] = value
		return nil
	}

	for _, set := range sets {
		key, value, found := strings.Cut(set, "=")
		if !found {
			return nil, fmt.Errorf("invalid --parameters %q: expected namespace.name=value (e.g. %s)", set, example)
		}
		if err := add("--parameters", set, key, &value); err != nil {
			return nil, err
		}
	}

	for _, reset := range resets {
		if err := add("--reset", reset, reset, nil); err != nil {
			return nil, err
		}
	}

	return changes, nil
}

type Parameter struct {
	Namespace string `header:"namespace" json:"namespace"`
	Name      string `header:"name" json:"name"`
	Value     string `header:"value" json:"value"`
	Default   string `header:"default" json:"default_value"`
	Type      string `header:"type" json:"parameter_type"`
	Override  bool   `header:"override" json:"override"`

	orig *ps.VitessParameter
}

func ToParameters(parameters []*ps.VitessParameter) []*Parameter {
	out := make([]*Parameter, 0, len(parameters))
	for _, param := range parameters {
		out = append(out, &Parameter{
			Namespace: param.Component,
			Name:      param.Name,
			Value:     stringValue(param.Value),
			Default:   stringValue(param.DefaultValue),
			Type:      param.ParameterType,
			Override:  param.Override,
			orig:      param,
		})
	}
	return out
}

func (p *Parameter) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(p.orig, "", "  ")
}

func (p *Parameter) MarshalCSVValue() interface{} {
	return []*Parameter{p}
}

type ConfigChange struct {
	ID        string `header:"id" json:"id"`
	Namespace string `header:"namespace" json:"change_type"`
	State     string `header:"state" json:"state"`
	Changes   string `header:"changes" json:"changes"`
	CreatedAt string `header:"created at" json:"created_at"`

	orig *ps.VitessConfigChange
}

func ToConfigChange(change *ps.VitessConfigChange) *ConfigChange {
	return &ConfigChange{
		ID:        change.ID,
		Namespace: change.ChangeType,
		State:     change.State,
		Changes:   formatChanges(change),
		CreatedAt: change.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:      change,
	}
}

func ToConfigChanges(changes []*ps.VitessConfigChange) []*ConfigChange {
	out := make([]*ConfigChange, 0, len(changes))
	for _, change := range changes {
		out = append(out, ToConfigChange(change))
	}
	return out
}

func (c *ConfigChange) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(c.orig, "", "  ")
}

func (c *ConfigChange) MarshalCSVValue() interface{} {
	return []*ConfigChange{c}
}

func formatChanges(change *ps.VitessConfigChange) string {
	seen := make(map[string]struct{})
	for name := range change.PreviousOptions {
		seen[name] = struct{}{}
	}
	for name := range change.NewOptions {
		seen[name] = struct{}{}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		before := optionDisplayValue(change.PreviousOptions[name])
		after := optionDisplayValue(change.NewOptions[name])
		if before == after {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s → %s", name, before, after))
	}
	return strings.Join(parts, ", ")
}

func optionDisplayValue(value *string) string {
	if value == nil {
		return "(default)"
	}
	return *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
