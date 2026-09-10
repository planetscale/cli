package branch

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/planetscale"
)

type dataTopologyTreeOutput struct {
	Database string                `json:"database"`
	Branch   string                `json:"branch"`
	SyncedAt *time.Time            `json:"synced_at"`
	Tree     *dataTopologyTreeNode `json:"tree"`
}

type dataTopologyTreeNode struct {
	Name     string                  `json:"name"`
	Children []*dataTopologyTreeNode `json:"children,omitempty"`
}

type dataTopologyDocument struct {
	AuthoritativeShardGroup string                            `json:"authoritative_shard_group"`
	ShardIndexes            map[string]dataTopologyShardIndex `json:"shard_indexes"`
	ShardGroups             []dataTopologyShardGroup          `json:"shard_groups"`
	Databases               map[string]dataTopologyDatabase   `json:"databases"`
}

type dataTopologyShardIndex struct {
	Type        string            `json:"type"`
	Columns     []string          `json:"columns,omitempty"`
	IndexParams map[string]string `json:"index_params,omitempty"`
}

type dataTopologyShardGroup struct {
	UID               string                 `json:"uid"`
	Name              string                 `json:"name,omitempty"`
	DefaultShardIndex string                 `json:"default_shard_index,omitempty"`
	KeyRanges         []dataTopologyKeyRange `json:"key_ranges,omitempty"`
}

type dataTopologyKeyRange struct {
	ShardUID string `json:"shard_uid"`
	Start    string `json:"start,omitempty"`
	End      string `json:"end,omitempty"`
}

type dataTopologyDatabase struct {
	AuthoritativeShardGroup string                             `json:"authoritative_shard_group,omitempty"`
	DefaultShardGroup       string                             `json:"default_shard_group,omitempty"`
	GlobalSecondaryIndexes  map[string]dataTopologyGlobalIndex `json:"global_secondary_indexes,omitempty"`
	Schemas                 map[string]dataTopologySchema      `json:"schemas,omitempty"`
}

type dataTopologySchema struct {
	DefaultShardGroup string                                `json:"default_shard_group,omitempty"`
	Tables            map[string]dataTopologyTable          `json:"tables,omitempty"`
	Sequences         []dataTopologySequence                `json:"sequences,omitempty"`
	ReferenceTables   map[string]dataTopologyReferenceTable `json:"reference_tables,omitempty"`
}

type dataTopologySequence struct {
	Name          string `json:"name"`
	ShardGroupUID string `json:"shard_group_uid,omitempty"`
}

type dataTopologyTable struct {
	ShardGroupUID    string                       `json:"shard_group_uid,omitempty"`
	PrimaryIndexes   []dataTopologyPrimaryIndex   `json:"primary_indexes,omitempty"`
	SecondaryIndexes []dataTopologySecondaryIndex `json:"secondary_indexes,omitempty"`
}

type dataTopologyPrimaryIndex struct {
	ShardIndex string   `json:"shard_index,omitempty"`
	Columns    []string `json:"columns,omitempty"`
	IgnoreNull bool     `json:"ignore_null,omitempty"`
}

type dataTopologySecondaryIndex struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns,omitempty"`
}

type dataTopologyReferenceTable struct {
	ShardGroupUIDs []string `json:"shard_group_uids,omitempty"`
}

type dataTopologyGlobalIndex struct {
	Schema         string   `json:"schema,omitempty"`
	Table          string   `json:"table"`
	OwnerTable     string   `json:"owner_table"`
	Columns        []string `json:"columns"`
	OwnerPKColumns []string `json:"owner_pk_columns,omitempty"`
	Unique         bool     `json:"unique,omitempty"`
	IgnoreNull     bool     `json:"ignore_null,omitempty"`
	Enabled        bool     `json:"enabled,omitempty"`
}

func buildDataTopologyTree(organization, database, branch string, dataTopology *planetscale.DataTopology) (*dataTopologyTreeOutput, error) {
	document, err := parseDataTopologyDocument(dataTopology)
	if err != nil {
		return nil, err
	}

	root := &dataTopologyTreeNode{Name: organization + "/" + database + "/" + branch}
	if len(document.Databases) > 0 {
		root.Children = append(root.Children, databasesTree(document))
	}
	if len(document.ShardGroups) > 0 {
		root.Children = append(root.Children, shardGroupsTree(document))
	}
	if len(document.ShardIndexes) > 0 {
		root.Children = append(root.Children, shardIndexesTree(document))
	}

	return newDataTopologyTreeOutput(database, branch, dataTopology, root), nil
}

func buildDataTopologyShardsTree(organization, database, branch string, dataTopology *planetscale.DataTopology) (*dataTopologyTreeOutput, error) {
	document, err := parseDataTopologyDocument(dataTopology)
	if err != nil {
		return nil, err
	}

	root := &dataTopologyTreeNode{Name: organization + "/" + database + "/" + branch}
	root.Children = shardNodes(document)
	return newDataTopologyTreeOutput(database, branch, dataTopology, root), nil
}

func parseDataTopologyDocument(dataTopology *planetscale.DataTopology) (*dataTopologyDocument, error) {
	var document dataTopologyDocument
	if err := json.Unmarshal(dataTopology.DataTopology, &document); err != nil {
		return nil, fmt.Errorf("reading data topology: %w", err)
	}
	return &document, nil
}

func newDataTopologyTreeOutput(database, branch string, dataTopology *planetscale.DataTopology, root *dataTopologyTreeNode) *dataTopologyTreeOutput {
	return &dataTopologyTreeOutput{
		Database: database,
		Branch:   branch,
		SyncedAt: dataTopology.SyncedAt,
		Tree:     root,
	}
}

type dataTopologyShardPlacement struct {
	Group    dataTopologyShardGroup
	KeyRange dataTopologyKeyRange
}

type dataTopologyHostedRelation struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
}

type dataTopologyHostedData struct {
	Tables          []dataTopologyHostedRelation `json:"tables,omitempty"`
	ReferenceTables []dataTopologyHostedRelation `json:"reference_tables,omitempty"`
	Sequences       []dataTopologyHostedRelation `json:"sequences,omitempty"`
}

func shardNodes(document *dataTopologyDocument) []*dataTopologyTreeNode {
	var nodes []*dataTopologyTreeNode
	placements := shardPlacementsByUID(document)
	hostedData := dataHostedByShardGroup(document)
	for _, shardUID := range sortedKeys(placements) {
		shardNode := &dataTopologyTreeNode{Name: shardUID}
		for _, placement := range placements[shardUID] {
			label := placement.Group.UID + " " + formatKeyRangeBounds(placement.KeyRange)
			if placement.Group.UID == document.AuthoritativeShardGroup {
				label += " [authoritative]"
			}
			groupNode := &dataTopologyTreeNode{Name: label}
			for _, data := range hostedData[placement.Group.UID].labels() {
				groupNode.add(data)
			}
			shardNode.Children = append(shardNode.Children, groupNode)
		}
		nodes = append(nodes, shardNode)
	}
	return nodes
}

func shardPlacementsByUID(document *dataTopologyDocument) map[string][]dataTopologyShardPlacement {
	placements := make(map[string][]dataTopologyShardPlacement)
	for _, group := range document.ShardGroups {
		for _, keyRange := range group.KeyRanges {
			placements[keyRange.ShardUID] = append(placements[keyRange.ShardUID], dataTopologyShardPlacement{
				Group:    group,
				KeyRange: keyRange,
			})
		}
	}

	for shardUID, shardPlacements := range placements {
		slices.SortFunc(shardPlacements, func(a, b dataTopologyShardPlacement) int {
			if result := strings.Compare(a.Group.UID, b.Group.UID); result != 0 {
				return result
			}
			if result := strings.Compare(a.KeyRange.Start, b.KeyRange.Start); result != 0 {
				return result
			}
			return strings.Compare(a.KeyRange.End, b.KeyRange.End)
		})
		placements[shardUID] = shardPlacements
	}
	return placements
}

func dataHostedByShardGroup(document *dataTopologyDocument) map[string]dataTopologyHostedData {
	hostedData := make(map[string]dataTopologyHostedData)
	for _, databaseName := range sortedKeys(document.Databases) {
		database := document.Databases[databaseName]
		for _, schemaName := range sortedKeys(database.Schemas) {
			schema := database.Schemas[schemaName]
			for _, tableName := range sortedKeys(schema.Tables) {
				group, _ := resolveTableShardGroup(schemaName, schema.Tables[tableName], schema, database, document.AuthoritativeShardGroup)
				if group != "<unresolved>" {
					data := hostedData[group]
					data.Tables = append(data.Tables, dataTopologyHostedRelation{Database: databaseName, Schema: schemaName, Name: tableName})
					hostedData[group] = data
				}
			}
			for _, sequence := range schema.Sequences {
				group, _ := resolveSequenceShardGroup(sequence, database, document.AuthoritativeShardGroup)
				if group != "<unresolved>" {
					data := hostedData[group]
					data.Sequences = append(data.Sequences, dataTopologyHostedRelation{Database: databaseName, Schema: schemaName, Name: sequence.Name})
					hostedData[group] = data
				}
			}
			for _, tableName := range sortedKeys(schema.ReferenceTables) {
				for _, group := range resolveReferenceTableShardGroups(schema.ReferenceTables[tableName], schema, database) {
					data := hostedData[group]
					data.ReferenceTables = append(data.ReferenceTables, dataTopologyHostedRelation{Database: databaseName, Schema: schemaName, Name: tableName})
					hostedData[group] = data
				}
			}
		}
	}

	for group, data := range hostedData {
		data.Tables = sortAndCompactHostedRelations(data.Tables)
		data.ReferenceTables = sortAndCompactHostedRelations(data.ReferenceTables)
		data.Sequences = sortAndCompactHostedRelations(data.Sequences)
		hostedData[group] = data
	}
	return hostedData
}

func sortAndCompactHostedRelations(relations []dataTopologyHostedRelation) []dataTopologyHostedRelation {
	slices.SortFunc(relations, func(a, b dataTopologyHostedRelation) int {
		if result := strings.Compare(a.Database, b.Database); result != 0 {
			return result
		}
		if result := strings.Compare(a.Schema, b.Schema); result != 0 {
			return result
		}
		return strings.Compare(a.Name, b.Name)
	})
	return slices.Compact(relations)
}

func (d dataTopologyHostedData) labels() []string {
	labels := make([]string, 0, len(d.Tables)+len(d.ReferenceTables)+len(d.Sequences))
	for _, table := range d.Tables {
		labels = append(labels, table.path())
	}
	for _, table := range d.ReferenceTables {
		labels = append(labels, table.path()+" [reference]")
	}
	for _, sequence := range d.Sequences {
		labels = append(labels, sequence.path()+" [sequence]")
	}
	slices.Sort(labels)
	return labels
}

func (r dataTopologyHostedRelation) path() string {
	return r.Database + "." + r.Schema + "." + r.Name
}

func databasesTree(document *dataTopologyDocument) *dataTopologyTreeNode {
	root := &dataTopologyTreeNode{Name: "databases"}
	for _, databaseName := range sortedKeys(document.Databases) {
		database := document.Databases[databaseName]
		node := &dataTopologyTreeNode{Name: databaseName}
		if database.AuthoritativeShardGroup != "" {
			node.add("authoritative shard group → " + database.AuthoritativeShardGroup)
		}
		if database.DefaultShardGroup != "" {
			node.add("default shard group → " + database.DefaultShardGroup)
		}
		if len(database.GlobalSecondaryIndexes) > 0 {
			node.Children = append(node.Children, globalIndexesTree(database.GlobalSecondaryIndexes))
		}
		if len(database.Schemas) > 0 {
			node.Children = append(node.Children, schemasTree(database, document.AuthoritativeShardGroup))
		}
		root.Children = append(root.Children, node)
	}
	return root
}

func globalIndexesTree(indexes map[string]dataTopologyGlobalIndex) *dataTopologyTreeNode {
	root := &dataTopologyTreeNode{Name: "global secondary indexes"}
	for _, name := range sortedKeys(indexes) {
		index := indexes[name]
		schema := index.Schema
		if schema == "" {
			schema = "public"
		}
		label := fmt.Sprintf("%s: %s(%s) → %s.%s", name, index.OwnerTable, strings.Join(index.Columns, ", "), schema, index.Table)
		var attributes []string
		if index.Unique {
			attributes = append(attributes, "unique")
		}
		if index.Enabled {
			attributes = append(attributes, "enabled")
		}
		if len(attributes) > 0 {
			label += " [" + strings.Join(attributes, ", ") + "]"
		}
		root.add(label)
	}
	return root
}

func schemasTree(database dataTopologyDatabase, clusterAuthoritativeGroup string) *dataTopologyTreeNode {
	root := &dataTopologyTreeNode{Name: "schemas"}
	for _, schemaName := range sortedKeys(database.Schemas) {
		schema := database.Schemas[schemaName]
		node := &dataTopologyTreeNode{Name: schemaName}
		if schema.DefaultShardGroup != "" {
			node.add("default shard group → " + schema.DefaultShardGroup)
		}
		if len(schema.Tables) > 0 {
			node.Children = append(node.Children, tablesTree(schemaName, schema, database, clusterAuthoritativeGroup))
		}
		if len(schema.Sequences) > 0 {
			node.Children = append(node.Children, sequencesTree(schema, database, clusterAuthoritativeGroup))
		}
		if len(schema.ReferenceTables) > 0 {
			node.Children = append(node.Children, referenceTablesTree(schema, database))
		}
		root.Children = append(root.Children, node)
	}
	return root
}

func sequencesTree(schema dataTopologySchema, database dataTopologyDatabase, clusterAuthoritativeGroup string) *dataTopologyTreeNode {
	root := &dataTopologyTreeNode{Name: "sequences"}
	sequences := slices.Clone(schema.Sequences)
	slices.SortFunc(sequences, func(a, b dataTopologySequence) int {
		return strings.Compare(a.Name, b.Name)
	})
	for _, sequence := range sequences {
		group, source := resolveSequenceShardGroup(sequence, database, clusterAuthoritativeGroup)
		label := sequence.Name + " → " + group
		if source != "" {
			label += " (" + source + ")"
		}
		root.add(label)
	}
	return root
}

func resolveSequenceShardGroup(sequence dataTopologySequence, database dataTopologyDatabase, clusterAuthoritativeGroup string) (string, string) {
	if sequence.ShardGroupUID != "" {
		return sequence.ShardGroupUID, "sequence"
	}
	if database.AuthoritativeShardGroup != "" {
		return database.AuthoritativeShardGroup, "database authoritative"
	}
	if clusterAuthoritativeGroup != "" {
		return clusterAuthoritativeGroup, "cluster authoritative"
	}
	return "<unresolved>", ""
}

func tablesTree(schemaName string, schema dataTopologySchema, database dataTopologyDatabase, clusterAuthoritativeGroup string) *dataTopologyTreeNode {
	root := &dataTopologyTreeNode{Name: "tables"}
	for _, tableName := range sortedKeys(schema.Tables) {
		table := schema.Tables[tableName]
		group, source := resolveTableShardGroup(schemaName, table, schema, database, clusterAuthoritativeGroup)
		label := tableName + " → " + group
		if source != "" && source != "table" {
			label += " (" + source + ")"
		}
		node := &dataTopologyTreeNode{Name: label}

		for _, index := range table.PrimaryIndexes {
			label := "primary index"
			if index.ShardIndex != "" {
				label += " → " + index.ShardIndex
			}
			if len(index.Columns) > 0 {
				label += " (" + strings.Join(index.Columns, ", ") + ")"
			}
			if index.IgnoreNull {
				label += " [ignore null]"
			}
			node.add(label)
		}

		for _, index := range table.SecondaryIndexes {
			label := "secondary index → " + index.Name
			if len(index.Columns) > 0 {
				label += " (" + strings.Join(index.Columns, ", ") + ")"
			}
			if globalIndex, ok := database.GlobalSecondaryIndexes[index.Name]; ok {
				schema := globalIndex.Schema
				if schema == "" {
					schema = "public"
				}
				label += " → " + schema + "." + globalIndex.Table
			}
			node.add(label)
		}

		root.Children = append(root.Children, node)
	}
	return root
}

func resolveTableShardGroup(schemaName string, table dataTopologyTable, schema dataTopologySchema, database dataTopologyDatabase, clusterAuthoritativeGroup string) (string, string) {
	if table.ShardGroupUID != "" {
		return table.ShardGroupUID, "table"
	}
	if schema.DefaultShardGroup != "" {
		return schema.DefaultShardGroup, "schema default"
	}
	if database.DefaultShardGroup != "" {
		return database.DefaultShardGroup, "database default"
	}
	if isSystemSchema(schemaName) {
		if database.AuthoritativeShardGroup != "" {
			return database.AuthoritativeShardGroup, "database authoritative"
		}
		if clusterAuthoritativeGroup != "" {
			return clusterAuthoritativeGroup, "cluster authoritative"
		}
	}
	return "<unresolved>", ""
}

func referenceTablesTree(schema dataTopologySchema, database dataTopologyDatabase) *dataTopologyTreeNode {
	root := &dataTopologyTreeNode{Name: "reference tables"}
	for _, tableName := range sortedKeys(schema.ReferenceTables) {
		groups := resolveReferenceTableShardGroups(schema.ReferenceTables[tableName], schema, database)
		label := tableName + " → "
		if len(groups) == 0 {
			label += "<unresolved>"
		} else {
			label += strings.Join(groups, ", ")
		}
		root.add(label)
	}
	return root
}

func resolveReferenceTableShardGroups(referenceTable dataTopologyReferenceTable, schema dataTopologySchema, database dataTopologyDatabase) []string {
	groups := slices.Clone(referenceTable.ShardGroupUIDs)
	if len(groups) == 0 {
		if schema.DefaultShardGroup != "" {
			groups = []string{schema.DefaultShardGroup}
		} else if database.DefaultShardGroup != "" {
			groups = []string{database.DefaultShardGroup}
		}
	}
	slices.Sort(groups)
	return slices.Compact(groups)
}

func shardGroupsTree(document *dataTopologyDocument) *dataTopologyTreeNode {
	root := &dataTopologyTreeNode{Name: "shard groups"}
	groups := slices.Clone(document.ShardGroups)
	slices.SortFunc(groups, func(a, b dataTopologyShardGroup) int {
		return strings.Compare(a.UID, b.UID)
	})

	for _, group := range groups {
		label := group.UID
		if group.Name != "" && group.Name != group.UID {
			label += " (" + group.Name + ")"
		}
		if group.DefaultShardIndex != "" {
			label += " → " + group.DefaultShardIndex
		}
		if group.UID == document.AuthoritativeShardGroup {
			label += " [authoritative]"
		}
		node := &dataTopologyTreeNode{Name: label}
		if len(group.KeyRanges) > 0 {
			ranges := &dataTopologyTreeNode{Name: "key ranges"}
			for _, keyRange := range group.KeyRanges {
				ranges.add(formatKeyRange(keyRange))
			}
			node.Children = append(node.Children, ranges)
		}
		root.Children = append(root.Children, node)
	}
	return root
}

func shardIndexesTree(document *dataTopologyDocument) *dataTopologyTreeNode {
	root := &dataTopologyTreeNode{Name: "shard indexes"}
	for _, name := range sortedKeys(document.ShardIndexes) {
		index := document.ShardIndexes[name]
		label := name + ": " + index.Type
		if len(index.Columns) > 0 {
			label += "(" + strings.Join(index.Columns, ", ") + ")"
		}
		if len(index.IndexParams) > 0 {
			var params []string
			for _, key := range sortedKeys(index.IndexParams) {
				params = append(params, key+"="+index.IndexParams[key])
			}
			label += " [" + strings.Join(params, ", ") + "]"
		}
		root.add(label)
	}
	return root
}

func formatKeyRange(keyRange dataTopologyKeyRange) string {
	return fmt.Sprintf("%s → %s", formatKeyRangeBounds(keyRange), keyRange.ShardUID)
}

func formatKeyRangeBounds(keyRange dataTopologyKeyRange) string {
	start := keyRange.Start
	if start == "" {
		start = "-∞"
	}
	end := keyRange.End
	if end == "" {
		end = "+∞"
	}
	return fmt.Sprintf("[%s, %s)", start, end)
}

func isSystemSchema(name string) bool {
	return name == "pg_catalog" || name == "information_schema" || name == "pg_toast"
}

func renderDataTopologyTree(root *dataTopologyTreeNode) string {
	var output strings.Builder
	output.WriteString(root.Name)
	output.WriteByte('\n')
	for i, child := range root.Children {
		renderDataTopologyTreeNode(&output, child, "", i == len(root.Children)-1)
	}
	return output.String()
}

func renderDataTopologyTreeNode(output *strings.Builder, node *dataTopologyTreeNode, prefix string, last bool) {
	output.WriteString(prefix)
	if last {
		output.WriteString("└── ")
	} else {
		output.WriteString("├── ")
	}
	output.WriteString(node.Name)
	output.WriteByte('\n')

	childPrefix := prefix
	if last {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}
	for i, child := range node.Children {
		renderDataTopologyTreeNode(output, child, childPrefix, i == len(node.Children)-1)
	}
}

func (n *dataTopologyTreeNode) add(name string) {
	n.Children = append(n.Children, &dataTopologyTreeNode{Name: name})
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
