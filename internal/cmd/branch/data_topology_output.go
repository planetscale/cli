package branch

import (
	"time"

	"github.com/planetscale/cli/internal/planetscale"
)

type dataTopologyListOutput struct {
	Organization            string                            `json:"organization"`
	Database                string                            `json:"database"`
	Branch                  string                            `json:"branch"`
	SyncedAt                *time.Time                        `json:"synced_at"`
	AuthoritativeShardGroup string                            `json:"authoritative_shard_group"`
	Databases               map[string]dataTopologyDatabase   `json:"databases"`
	ShardGroups             []dataTopologyShardGroup          `json:"shard_groups"`
	ShardIndexes            map[string]dataTopologyShardIndex `json:"shard_indexes"`
}

func buildDataTopologyListOutput(organization, database, branch string, dataTopology *planetscale.DataTopology) (*dataTopologyListOutput, error) {
	document, err := parseDataTopologyDocument(dataTopology)
	if err != nil {
		return nil, err
	}

	return &dataTopologyListOutput{
		Organization:            organization,
		Database:                database,
		Branch:                  branch,
		SyncedAt:                dataTopology.SyncedAt,
		AuthoritativeShardGroup: document.AuthoritativeShardGroup,
		Databases:               document.Databases,
		ShardGroups:             document.ShardGroups,
		ShardIndexes:            document.ShardIndexes,
	}, nil
}
