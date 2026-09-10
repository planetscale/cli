package admin

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

func AdminCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin <command>",
		Short: "Manage the admin config for a Neki database branch",
		Long:  "Manage the admin config for a Neki database branch.\n\nEach cluster has one admin. The admin is created and deleted with the cluster. This command is only supported for Neki databases.",
	}

	cmd.AddCommand(
		ChangesCmd(ch),
		ParametersCmd(ch),
		ShowCmd(ch),
		SizesCmd(ch),
		UpdateCmd(ch),
	)

	return cmd
}

type adminDisplay struct {
	ID        string `header:"id" json:"id"`
	Size      string `header:"size" json:"size"`
	State     string `header:"state" json:"state"`
	CreatedAt string `header:"created at" json:"created_at"`

	orig *ps.NekiAdmin
}

func (a *adminDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.orig)
}

func (a *adminDisplay) MarshalCSVValue() interface{} {
	return []*adminDisplay{a}
}

func toAdmin(admin *ps.NekiAdmin) *adminDisplay {
	return &adminDisplay{
		ID:        admin.ID,
		Size:      adminSize(admin),
		State:     admin.State,
		CreatedAt: admin.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:      admin,
	}
}

func adminSize(admin *ps.NekiAdmin) string {
	if admin.SKU != nil {
		if admin.SKU.DisplayName != "" {
			return admin.SKU.DisplayName
		}
		if admin.SKU.Name != "" {
			return admin.SKU.Name
		}
	}
	return admin.AdminSize
}

func changeSize(name, displayName string) string {
	if displayName != "" {
		return displayName
	}
	return name
}

type adminDetailDisplay struct {
	*adminDisplay

	parameters []*ps.NekiParameter
}

func (a *adminDetailDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		*ps.NekiAdmin
		Parameters []*ps.NekiParameter `json:"parameters"`
	}{NekiAdmin: a.orig, Parameters: a.parameters})
}

func toAdminDetail(admin *ps.NekiAdmin, parameters []*ps.NekiParameter) *adminDetailDisplay {
	return &adminDetailDisplay{adminDisplay: toAdmin(admin), parameters: parameters}
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
	orig      *ps.NekiAdminChange
}

func (c *changeDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(c.orig) }
func (c *changeDisplay) MarshalCSVValue() interface{} { return []*changeDisplay{c} }

func toChange(c *ps.NekiAdminChange) *changeDisplay {
	return &changeDisplay{
		ID:        c.ID,
		State:     c.State,
		Changes:   formatChangeSummary(c),
		CreatedAt: c.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:      c,
	}
}

func formatChangeSummary(change *ps.NekiAdminChange) string {
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

func toChanges(changes []*ps.NekiAdminChange) []*changeDisplay {
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

func changeDifferences(change *ps.NekiAdminChange) []changeDifference {
	differences := make([]changeDifference, 0)
	before := changeSize(change.PreviousAdminSize, change.PreviousAdminSizeDisplayName)
	after := changeSize(change.AdminSize, change.AdminSizeDisplayName)
	if before != after {
		differences = append(differences, changeDifference{Field: "Size", Before: before, After: after})
	}

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
		differences = append(differences, changeDifference{Field: "Parameter " + key, Before: before, After: after})
	}

	return differences
}
