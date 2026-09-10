package branch

import (
	"time"

	"github.com/planetscale/cli/internal/planetscale"
)

type dataTopologyShardsOutput struct {
	Organization            string                       `json:"organization"`
	Database                string                       `json:"database"`
	Branch                  string                       `json:"branch"`
	SyncedAt                *time.Time                   `json:"synced_at"`
	AuthoritativeShardGroup string                       `json:"authoritative_shard_group"`
	Shards                  map[string]dataTopologyShard `json:"shards"`
}

type dataTopologyShard struct {
	Placements []dataTopologyPlacement `json:"placements"`
}

type dataTopologyPlacement struct {
	ShardGroup        string                       `json:"shard_group"`
	ShardGroupName    string                       `json:"shard_group_name,omitempty"`
	Authoritative     bool                         `json:"authoritative"`
	KeyRange          dataTopologyKeyRangeOutput   `json:"key_range"`
	DefaultShardIndex string                       `json:"default_shard_index,omitempty"`
	Tables            []dataTopologyHostedRelation `json:"tables,omitempty"`
	ReferenceTables   []dataTopologyHostedRelation `json:"reference_tables,omitempty"`
	Sequences         []dataTopologyHostedRelation `json:"sequences,omitempty"`
}

type dataTopologyKeyRangeOutput struct {
	Start *string `json:"start"`
	End   *string `json:"end"`
}

func buildDataTopologyShardsOutput(organization, database, branch string, dataTopology *planetscale.DataTopology) (*dataTopologyShardsOutput, error) {
	document, err := parseDataTopologyDocument(dataTopology)
	if err != nil {
		return nil, err
	}

	output := &dataTopologyShardsOutput{
		Organization:            organization,
		Database:                database,
		Branch:                  branch,
		SyncedAt:                dataTopology.SyncedAt,
		AuthoritativeShardGroup: document.AuthoritativeShardGroup,
		Shards:                  make(map[string]dataTopologyShard),
	}
	placements := shardPlacementsByUID(document)
	hostedData := dataHostedByShardGroup(document)
	for _, shardUID := range sortedKeys(placements) {
		shard := dataTopologyShard{}
		for _, placement := range placements[shardUID] {
			data := hostedData[placement.Group.UID]
			shard.Placements = append(shard.Placements, dataTopologyPlacement{
				ShardGroup:        placement.Group.UID,
				ShardGroupName:    placement.Group.Name,
				Authoritative:     placement.Group.UID == document.AuthoritativeShardGroup,
				KeyRange:          dataTopologyKeyRangeOutput{Start: optionalTopologyBound(placement.KeyRange.Start), End: optionalTopologyBound(placement.KeyRange.End)},
				DefaultShardIndex: placement.Group.DefaultShardIndex,
				Tables:            data.Tables,
				ReferenceTables:   data.ReferenceTables,
				Sequences:         data.Sequences,
			})
		}
		output.Shards[shardUID] = shard
	}
	return output, nil
}

func optionalTopologyBound(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
