package shard

import (
	"encoding/json"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/spf13/cobra"
)

func ShardCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shard <command>",
		Short: "Manage shards for a Neki database branch",
		Long:  "Manage shards for a Neki database branch.\n\nThis command is only supported for Neki databases.",
	}

	cmd.AddCommand(
		AssignCmd(ch),
		CreateCmd(ch),
		DeleteCmd(ch),
		ListCmd(ch),
		ShowCmd(ch),
		UpdateCmd(ch),
	)

	return cmd
}

type shardDisplay struct {
	ID                   string `header:"id" json:"id"`
	Name                 string `header:"name" json:"name"`
	DisplayName          string `header:"display name" json:"display_name"`
	ConfigurationProfile string `header:"configuration profile" json:"configuration_profile"`
	Ready                bool   `header:"ready" json:"ready"`
	Authoritative        bool   `header:"authoritative" json:"authoritative"`
	CreatedAt            string `header:"created at" json:"created_at"`

	orig *ps.NekiShard
}

func (s *shardDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.orig)
}

func (s *shardDisplay) MarshalCSVValue() interface{} {
	return []*shardDisplay{s}
}

func toShard(shard *ps.NekiShard) *shardDisplay {
	displayName := ""
	if shard.DisplayName != nil {
		displayName = *shard.DisplayName
	}

	return &shardDisplay{
		ID:                   shard.ID,
		Name:                 shard.Name,
		DisplayName:          displayName,
		ConfigurationProfile: shard.ConfigurationProfile,
		Ready:                shard.Ready,
		Authoritative:        shard.Authoritative,
		CreatedAt:            shard.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:                 shard,
	}
}

func toShards(shards []*ps.NekiShard) []*shardDisplay {
	result := make([]*shardDisplay, 0, len(shards))
	for _, shard := range shards {
		result = append(result, toShard(shard))
	}
	return result
}

type shardCreationDisplay struct {
	Created  int    `header:"created" json:"created"`
	ShardIDs string `header:"shard ids" json:"shard_ids"`

	orig *ps.CreateNekiShardsResponse
}

func (s *shardCreationDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.orig)
}

func (s *shardCreationDisplay) MarshalCSVValue() interface{} {
	return []*shardCreationDisplay{s}
}

func toShardCreation(response *ps.CreateNekiShardsResponse) *shardCreationDisplay {
	return &shardCreationDisplay{
		Created:  response.Created,
		ShardIDs: strings.Join(response.ShardIDs, ", "),
		orig:     response,
	}
}

type shardAssignmentDisplay struct {
	ID     string `header:"id" json:"id"`
	Status string `header:"status" json:"status"`
	Error  string `header:"error" json:"error,omitempty"`

	orig *ps.NekiShardAssignment
}

func (s *shardAssignmentDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.orig)
}

func toShardAssignments(assignments []*ps.NekiShardAssignment) []*shardAssignmentDisplay {
	result := make([]*shardAssignmentDisplay, 0, len(assignments))
	for _, assignment := range assignments {
		errorText := ""
		if assignment.Error != nil {
			encoded, err := json.Marshal(assignment.Error)
			if err == nil {
				errorText = string(encoded)
			}
		}
		result = append(result, &shardAssignmentDisplay{
			ID:     assignment.ID,
			Status: assignment.Status,
			Error:  errorText,
			orig:   assignment,
		})
	}
	return result
}
