package configprofile

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

func ConfigProfileCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config-profile <command>",
		Short: "Manage configuration profiles for a Neki database branch",
		Long:  "Manage configuration profiles for a Neki database branch.\n\nThis command is only supported for Neki databases.",
	}
	cmd.AddCommand(
		ChangesCmd(ch), CreateCmd(ch), DefaultCmd(ch), DeleteCmd(ch), ExtensionsCmd(ch),
		ListCmd(ch), MaintenanceCmd(ch), ParametersCmd(ch), SetDefaultCmd(ch), ShowCmd(ch), UpdateCmd(ch),
	)
	return cmd
}

type profileDisplay struct {
	Name            string `header:"name" json:"name"`
	State           string `header:"state" json:"state"`
	Default         bool   `header:"default" json:"default"`
	Architecture    string `header:"architecture" json:"architecture"`
	ClusterSize     string `header:"cluster size" json:"cluster_size"`
	Replicas        int    `header:"replicas" json:"replicas"`
	Shards          int    `header:"shards" json:"shards"`
	PostgresVersion string `header:"postgres version" json:"postgres_version"`
	MinStorage      string `header:"min storage" json:"min_storage"`
	MaxStorage      string `header:"max storage" json:"max_storage"`
	Autoscaling     string `header:"storage autoscaling" json:"storage_autoscaling"`
	CreatedAt       string `header:"created at" json:"created_at"`
	orig            *ps.NekiShardConfigurationProfile
}

func (p *profileDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(p.orig) }
func (p *profileDisplay) MarshalCSVValue() interface{} { return []*profileDisplay{p} }

func toProfile(p *ps.NekiShardConfigurationProfile) *profileDisplay {
	return &profileDisplay{
		Name:            p.Name,
		State:           p.State,
		Default:         p.Default,
		Architecture:    p.Architecture,
		ClusterSize:     p.ClusterSize,
		Replicas:        p.Replicas,
		Shards:          p.Shards,
		PostgresVersion: fmt.Sprintf("%d.%d", p.PostgresMajorVersion, p.PostgresMinorVersion),
		MinStorage:      storageBytes(p.Storage, func(s *ps.NekiStorage) *int64 { return s.MinimumStorageBytes }),
		MaxStorage:      storageBytes(p.Storage, func(s *ps.NekiStorage) *int64 { return s.MaximumStorageBytes }),
		Autoscaling:     storageAutoscaling(p.Storage),
		CreatedAt:       p.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:            p,
	}
}

func toProfiles(profiles []*ps.NekiShardConfigurationProfile) []*profileDisplay {
	out := make([]*profileDisplay, 0, len(profiles))
	for _, profile := range profiles {
		out = append(out, toProfile(profile))
	}
	return out
}

type parameterDisplay struct {
	Namespace string `header:"namespace" json:"namespace"`
	Name      string `header:"name" json:"name"`
	Value     string `header:"value" json:"value"`
	Default   string `header:"default" json:"default_value"`
	Type      string `header:"type" json:"parameter_type"`
	Restart   bool   `header:"restart" json:"restart"`
	Required  bool   `header:"required" json:"required"`
	orig      *ps.NekiParameter
}

func (p *parameterDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(p.orig) }

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
func toParameters(parameters []*ps.NekiParameter) []*parameterDisplay {
	out := make([]*parameterDisplay, 0, len(parameters))
	for _, p := range parameters {
		out = append(out, &parameterDisplay{
			Namespace: p.Namespace,
			Name:      p.Name,
			Value:     formatValue(p.Value),
			Default:   formatValue(p.DefaultValue),
			Type:      p.ParameterType,
			Restart:   p.Restart,
			Required:  p.Required,
			orig:      p,
		})
	}
	return out
}

type extensionDisplay struct {
	Name       string `header:"name" json:"name"`
	Enabled    bool   `header:"enabled" json:"enabled"`
	Internal   bool   `header:"internal" json:"internal"`
	Loader     string `header:"loader" json:"loader"`
	Parameters int    `header:"parameters" json:"parameters"`
	orig       *ps.NekiExtension
}

func (e *extensionDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(e.orig) }

func toExtension(e *ps.NekiExtension) *extensionDisplay {
	return &extensionDisplay{
		Name:       e.Name,
		Enabled:    e.Enabled,
		Internal:   e.Internal,
		Loader:     e.Loader,
		Parameters: len(e.Parameters),
		orig:       e,
	}
}

func toExtensions(extensions []*ps.NekiExtension) []*extensionDisplay {
	out := make([]*extensionDisplay, 0, len(extensions))
	for _, e := range extensions {
		out = append(out, toExtension(e))
	}
	return out
}

type changeDisplay struct {
	ID          string `header:"id" json:"id"`
	State       string `header:"state" json:"state"`
	Name        string `header:"name" json:"name"`
	ClusterSize string `header:"cluster size" json:"cluster_size"`
	Replicas    int    `header:"replicas" json:"replicas"`
	CreatedAt   string `header:"created at" json:"created_at"`
	orig        *ps.NekiShardConfigurationProfileChange
}

func (c *changeDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(c.orig) }
func (c *changeDisplay) MarshalCSVValue() interface{} { return []*changeDisplay{c} }

func toChange(c *ps.NekiShardConfigurationProfileChange) *changeDisplay {
	return &changeDisplay{
		ID:          c.ID,
		State:       c.State,
		Name:        c.Name,
		ClusterSize: c.ClusterSize,
		Replicas:    c.Replicas,
		CreatedAt:   c.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:        c,
	}
}

func toChanges(changes []*ps.NekiShardConfigurationProfileChange) []*changeDisplay {
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

func changeDifferences(change *ps.NekiShardConfigurationProfileChange) []changeDifference {
	differences := make([]changeDifference, 0)
	if change.PreviousName != change.Name {
		differences = append(differences, changeDifference{Field: "Name", Before: change.PreviousName, After: change.Name})
	}
	if change.PreviousClusterSize != change.ClusterSize {
		differences = append(differences, changeDifference{Field: "Cluster size", Before: change.PreviousClusterSize, After: change.ClusterSize})
	}
	if change.PreviousReplicas != change.Replicas {
		differences = append(differences, changeDifference{
			Field:  "Replicas",
			Before: strconv.Itoa(change.PreviousReplicas),
			After:  strconv.Itoa(change.Replicas),
		})
	}
	differences = appendStorageDifferences(differences, change.PreviousStorage, change.Storage)

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

func stringPointerIfChanged(cmd *cobra.Command, name, value string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}
func intPointerIfChanged(cmd *cobra.Command, name string, value int) *int {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}

func int64PointerIfChanged(cmd *cobra.Command, name string, value int64) *int64 {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}

func boolPointerIfChanged(cmd *cobra.Command, name string, value bool) *bool {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}

type storageFlags struct {
	minStorage, maxStorage, iops, throughput int64
	autoscaling                              bool
}

func bindStorageFlags(cmd *cobra.Command, flags *storageFlags) {
	cmd.Flags().Int64Var(&flags.minStorage, "min-storage", 0, "Minimum storage size in bytes")
	cmd.Flags().Int64Var(&flags.maxStorage, "max-storage", 0, "Maximum storage size in bytes for autoscaling")
	cmd.Flags().BoolVar(&flags.autoscaling, "storage-autoscaling", false, "Enable storage autoscaling")
	cmd.Flags().Int64Var(&flags.iops, "storage-iops", 0, "Storage IOPS")
	cmd.Flags().Int64Var(&flags.throughput, "storage-throughput", 0, "Storage throughput in MiB/s")
}

func storageFromFlags(cmd *cobra.Command, flags storageFlags) *ps.NekiStorage {
	storage := &ps.NekiStorage{
		MinimumStorageBytes:   int64PointerIfChanged(cmd, "min-storage", flags.minStorage),
		MaximumStorageBytes:   int64PointerIfChanged(cmd, "max-storage", flags.maxStorage),
		StorageAutoscaling:    boolPointerIfChanged(cmd, "storage-autoscaling", flags.autoscaling),
		StorageIOPS:           int64PointerIfChanged(cmd, "storage-iops", flags.iops),
		StorageThroughputMiBs: int64PointerIfChanged(cmd, "storage-throughput", flags.throughput),
	}
	if storage.MinimumStorageBytes == nil && storage.MaximumStorageBytes == nil && storage.StorageAutoscaling == nil && storage.StorageIOPS == nil && storage.StorageThroughputMiBs == nil {
		return nil
	}
	return storage
}

func storageFlagChanged(cmd *cobra.Command) bool {
	for _, name := range []string{"min-storage", "max-storage", "storage-autoscaling", "storage-iops", "storage-throughput"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func storageBytes(storage *ps.NekiStorage, value func(*ps.NekiStorage) *int64) string {
	if storage == nil {
		return ""
	}
	return formatOptionalInt64(value(storage))
}

func storageAutoscaling(storage *ps.NekiStorage) string {
	if storage == nil || storage.StorageAutoscaling == nil {
		return ""
	}
	if *storage.StorageAutoscaling {
		return "Yes"
	}
	return "No"
}

func formatOptionalInt64(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func formatOptionalBool(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "true"
	}
	return "false"
}

func appendStorageDifferences(differences []changeDifference, before, after *ps.NekiStorage) []changeDifference {
	pairs := []struct {
		field  string
		before string
		after  string
	}{
		{"Min storage", storageBytes(before, func(s *ps.NekiStorage) *int64 { return s.MinimumStorageBytes }), storageBytes(after, func(s *ps.NekiStorage) *int64 { return s.MinimumStorageBytes })},
		{"Max storage", storageBytes(before, func(s *ps.NekiStorage) *int64 { return s.MaximumStorageBytes }), storageBytes(after, func(s *ps.NekiStorage) *int64 { return s.MaximumStorageBytes })},
		{"Storage autoscaling", formatOptionalBool(storageAutoscalingPtr(before)), formatOptionalBool(storageAutoscalingPtr(after))},
		{"Storage IOPS", storageBytes(before, func(s *ps.NekiStorage) *int64 { return s.StorageIOPS }), storageBytes(after, func(s *ps.NekiStorage) *int64 { return s.StorageIOPS })},
		{"Storage throughput", storageBytes(before, func(s *ps.NekiStorage) *int64 { return s.StorageThroughputMiBs }), storageBytes(after, func(s *ps.NekiStorage) *int64 { return s.StorageThroughputMiBs })},
	}
	for _, pair := range pairs {
		if pair.before == pair.after {
			continue
		}
		if pair.before == "" {
			pair.before = "(default)"
		}
		if pair.after == "" {
			pair.after = "(default)"
		}
		differences = append(differences, changeDifference{Field: pair.field, Before: pair.before, After: pair.after})
	}
	return differences
}

func storageAutoscalingPtr(storage *ps.NekiStorage) *bool {
	if storage == nil {
		return nil
	}
	return storage.StorageAutoscaling
}
