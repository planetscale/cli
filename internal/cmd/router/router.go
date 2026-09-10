package router

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

func RouterCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "router <command>",
		Short: "Manage routers for a Neki database branch",
		Long:  "Manage routers for a Neki database branch.\n\nThis command is only supported for Neki databases.",
	}

	cmd.AddCommand(
		ChangesCmd(ch),
		CreateCmd(ch),
		DeleteCmd(ch),
		ListCmd(ch),
		ShowCmd(ch),
		SizesCmd(ch),
		UpdateCmd(ch),
	)

	return cmd
}

type routerDisplay struct {
	Name                 string `header:"name" json:"name"`
	Default              bool   `header:"default" json:"default"`
	Size                 string `header:"size" json:"size"`
	ReplicasPerCell      int    `header:"replicas per cell" json:"replicas_per_cell"`
	Autoscaling          bool   `header:"autoscaling" json:"autoscaling"`
	MaxReplicasPerCell   string `header:"max replicas per cell" json:"max_replicas_per_cell"`
	TargetCPUUtilization string `header:"target cpu utilization" json:"target_cpu_utilization"`
	State                string `header:"state" json:"state"`
	CreatedAt            string `header:"created at" json:"created_at"`

	orig *ps.NekiRouter
}

func (r *routerDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.orig)
}

func (r *routerDisplay) MarshalCSVValue() interface{} {
	return []*routerDisplay{r}
}

func toRouter(router *ps.NekiRouter) *routerDisplay {
	size := "-"
	if router.SKU != nil {
		size = router.SKU.Name
		if router.SKU.DisplayName != "" {
			size = router.SKU.DisplayName
		}
	}

	return &routerDisplay{
		Name:                 router.Name,
		Default:              router.Default,
		Size:                 size,
		ReplicasPerCell:      router.ReplicasPerCell,
		Autoscaling:          router.Autoscaling,
		MaxReplicasPerCell:   intOrDash(router.MaxReplicasPerCell),
		TargetCPUUtilization: intOrDash(router.TargetCPUUtilization),
		State:                router.State,
		CreatedAt:            router.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:                 router,
	}
}

func toRouters(routers []*ps.NekiRouter) []*routerDisplay {
	out := make([]*routerDisplay, 0, len(routers))
	for _, router := range routers {
		out = append(out, toRouter(router))
	}
	return out
}

type routerDetailDisplay struct {
	*routerDisplay

	parameters []*ps.NekiParameter
}

func (r *routerDetailDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		*ps.NekiRouter
		Parameters []*ps.NekiParameter `json:"parameters"`
	}{NekiRouter: r.orig, Parameters: r.parameters})
}

func toRouterDetail(router *ps.NekiRouter, parameters []*ps.NekiParameter) *routerDetailDisplay {
	return &routerDetailDisplay{routerDisplay: toRouter(router), parameters: parameters}
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

func intOrDash(v *int) string {
	if v == nil {
		return "-"
	}
	return strconv.Itoa(*v)
}

type changeDisplay struct {
	ID        string `header:"id" json:"id"`
	State     string `header:"state" json:"state"`
	Changes   string `header:"changes" json:"changes"`
	CreatedAt string `header:"created at" json:"created_at"`
	orig      *ps.NekiRouterChange
}

func (c *changeDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(c.orig) }
func (c *changeDisplay) MarshalCSVValue() interface{} { return []*changeDisplay{c} }

func toChange(c *ps.NekiRouterChange) *changeDisplay {
	return &changeDisplay{
		ID:        c.ID,
		State:     c.State,
		Changes:   formatChangeSummary(c),
		CreatedAt: c.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:      c,
	}
}

func formatChangeSummary(change *ps.NekiRouterChange) string {
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

func toChanges(changes []*ps.NekiRouterChange) []*changeDisplay {
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

func changeSize(name, displayName string) string {
	if displayName != "" {
		return displayName
	}
	return name
}

func changeDifferences(change *ps.NekiRouterChange) []changeDifference {
	differences := make([]changeDifference, 0)
	before := changeSize(change.PreviousRouterSize, change.PreviousRouterSizeDisplayName)
	after := changeSize(change.RouterSize, change.RouterSizeDisplayName)
	if before != after {
		differences = append(differences, changeDifference{Field: "Size", Before: before, After: after})
	}
	if change.PreviousReplicasPerCell != change.ReplicasPerCell {
		differences = append(differences, changeDifference{
			Field:  "Replicas per cell",
			Before: strconv.Itoa(change.PreviousReplicasPerCell),
			After:  strconv.Itoa(change.ReplicasPerCell),
		})
	}
	if optionalBoolChanged(change.PreviousAutoscaling, change.Autoscaling) {
		differences = append(differences, changeDifference{
			Field:  "Autoscaling",
			Before: formatOptionalBool(change.PreviousAutoscaling),
			After:  formatOptionalBool(change.Autoscaling),
		})
	}
	if optionalIntChanged(change.PreviousMaxReplicasPerCell, change.MaxReplicasPerCell) {
		differences = append(differences, changeDifference{
			Field:  "Max replicas per cell",
			Before: formatOptionalInt(change.PreviousMaxReplicasPerCell),
			After:  formatOptionalInt(change.MaxReplicasPerCell),
		})
	}
	if optionalIntChanged(change.PreviousTargetCPUUtilization, change.TargetCPUUtilization) {
		differences = append(differences, changeDifference{
			Field:  "Target CPU utilization",
			Before: formatOptionalInt(change.PreviousTargetCPUUtilization),
			After:  formatOptionalInt(change.TargetCPUUtilization),
		})
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

func optionalBoolChanged(before, after *bool) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return *before != *after
}

func optionalIntChanged(before, after *int) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return *before != *after
}

func formatOptionalBool(v *bool) string {
	if v == nil {
		return "(default)"
	}
	return strconv.FormatBool(*v)
}

func formatOptionalInt(v *int) string {
	if v == nil {
		return "(default)"
	}
	return strconv.Itoa(*v)
}
