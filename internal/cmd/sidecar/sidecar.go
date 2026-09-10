package sidecar

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/spf13/cobra"
)

func SidecarCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sidecar <command>",
		Short: "Manage sidecars for a Neki database branch",
		Long:  "Manage sidecars for a Neki database branch.\n\nEach configuration profile has one sidecar. Sidecars are created and deleted with their profile. This command is only supported for Neki databases.",
	}

	cmd.AddCommand(
		ChangesCmd(ch),
		ListCmd(ch),
		ParametersCmd(ch),
		ShowCmd(ch),
		UpdateCmd(ch),
	)

	return cmd
}

type sidecarDisplay struct {
	ID                   string `header:"id" json:"id"`
	ConfigurationProfile string `header:"configuration profile" json:"configuration_profile"`
	State                string `header:"state" json:"state"`
	CreatedAt            string `header:"created at" json:"created_at"`

	orig *ps.NekiSidecar
}

func (s *sidecarDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.orig)
}

func (s *sidecarDisplay) MarshalCSVValue() interface{} {
	return []*sidecarDisplay{s}
}

func toSidecar(sidecar *ps.NekiSidecar) *sidecarDisplay {
	return &sidecarDisplay{
		ID:                   sidecar.ID,
		ConfigurationProfile: sidecar.ConfigurationProfile,
		State:                sidecar.State,
		CreatedAt:            sidecar.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:                 sidecar,
	}
}

func toSidecars(sidecars []*ps.NekiSidecar) []*sidecarDisplay {
	out := make([]*sidecarDisplay, 0, len(sidecars))
	for _, sidecar := range sidecars {
		out = append(out, toSidecar(sidecar))
	}
	return out
}

type sidecarDetailDisplay struct {
	*sidecarDisplay

	parameters []*ps.NekiParameter
}

func (s *sidecarDetailDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		*ps.NekiSidecar
		Parameters []*ps.NekiParameter `json:"parameters"`
	}{NekiSidecar: s.orig, Parameters: s.parameters})
}

func toSidecarDetail(sidecar *ps.NekiSidecar, parameters []*ps.NekiParameter) *sidecarDetailDisplay {
	return &sidecarDetailDisplay{sidecarDisplay: toSidecar(sidecar), parameters: parameters}
}

type parameterDisplay struct {
	Namespace string `header:"namespace" json:"namespace"`
	Name      string `header:"name" json:"name"`
	Value     string `header:"value" json:"value"`
	Default   string `header:"default" json:"default_value"`
	Type      string `header:"type" json:"parameter_type"`
	Restart   bool   `header:"restart" json:"restart"`

	orig *ps.NekiParameter
}

func (p *parameterDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.orig)
}

func (p *parameterDisplay) MarshalCSVValue() interface{} {
	return []*parameterDisplay{p}
}

func toParameters(parameters []*ps.NekiParameter) []*parameterDisplay {
	out := make([]*parameterDisplay, 0, len(parameters))
	for _, parameter := range parameters {
		out = append(out, &parameterDisplay{
			Namespace: parameter.Namespace,
			Name:      parameter.Name,
			Value:     formatValue(parameter.Value),
			Default:   formatValue(parameter.DefaultValue),
			Type:      parameter.ParameterType,
			Restart:   parameter.Restart,
			orig:      parameter,
		})
	}
	return out
}

func formatValue(v interface{}) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", value)
	}
}

type changeDisplay struct {
	ID        string `header:"id" json:"id"`
	State     string `header:"state" json:"state"`
	Changes   string `header:"changes" json:"changes"`
	CreatedAt string `header:"created at" json:"created_at"`
	orig      *ps.NekiSidecarChange
}

func (c *changeDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(c.orig) }
func (c *changeDisplay) MarshalCSVValue() interface{} { return []*changeDisplay{c} }

func toChange(c *ps.NekiSidecarChange) *changeDisplay {
	return &changeDisplay{
		ID:        c.ID,
		State:     c.State,
		Changes:   formatChangeSummary(c),
		CreatedAt: c.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:      c,
	}
}

func formatChangeSummary(change *ps.NekiSidecarChange) string {
	differences := changeDifferences(change)
	if len(differences) == 0 {
		return ""
	}
	parts := make([]string, 0, len(differences))
	for _, difference := range differences {
		parts = append(parts, fmt.Sprintf("%s: %s → %s", difference.Field, difference.Before, difference.After))
	}
	return strings.Join(parts, ", ")
}

func toChanges(changes []*ps.NekiSidecarChange) []*changeDisplay {
	out := make([]*changeDisplay, 0, len(changes))
	for _, change := range changes {
		out = append(out, toChange(change))
	}
	return out
}

type changeDifference struct {
	Field  string
	Before string
	After  string
}

func changeDifferences(change *ps.NekiSidecarChange) []changeDifference {
	parameterKeys := make(map[string]struct{})
	for namespace, parameters := range change.PreviousParameters {
		for name := range parameters {
			parameterKeys[namespace+"."+name] = struct{}{}
		}
	}
	for namespace, parameters := range change.Parameters {
		for name := range parameters {
			parameterKeys[namespace+"."+name] = struct{}{}
		}
	}

	keys := make([]string, 0, len(parameterKeys))
	for key := range parameterKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	differences := make([]changeDifference, 0)
	for _, key := range keys {
		namespace, name, _ := strings.Cut(key, ".")
		before, hadBefore := change.PreviousParameters[namespace][name]
		after, hasAfter := change.Parameters[namespace][name]
		if hadBefore && hasAfter && before == after {
			continue
		}
		if !hadBefore {
			before = "(default)"
		}
		if !hasAfter {
			after = "(default)"
		}
		differences = append(differences, changeDifference{Field: key, Before: before, After: after})
	}

	return differences
}
