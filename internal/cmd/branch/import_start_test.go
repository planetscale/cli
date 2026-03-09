package branch

import (
	"reflect"
	"testing"

	"github.com/planetscale/cli/internal/postgres"
)

func TestParseTableFilters(t *testing.T) {
	tests := []struct {
		name              string
		includeTablesStr  string
		skipTablesStr     string
		schemas           []string
		wantIncludeTables []string
		wantSkipTables    []string
	}{
		{
			name:              "no filters",
			includeTablesStr:  "",
			skipTablesStr:     "",
			schemas:           []string{"public"},
			wantIncludeTables: []string(nil),
			wantSkipTables:    []string(nil),
		},
		{
			name:              "include tables without schema",
			includeTablesStr:  "users,posts",
			skipTablesStr:     "",
			schemas:           []string{"public"},
			wantIncludeTables: []string{"public.users", "public.posts"},
			wantSkipTables:    []string(nil),
		},
		{
			name:              "include tables with schema",
			includeTablesStr:  "public.users,app.posts",
			skipTablesStr:     "",
			schemas:           []string{"public"},
			wantIncludeTables: []string{"public.users", "app.posts"},
			wantSkipTables:    []string(nil),
		},
		{
			name:              "skip tables without schema",
			includeTablesStr:  "",
			skipTablesStr:     "temp,cache",
			schemas:           []string{"public"},
			wantIncludeTables: []string(nil),
			wantSkipTables:    []string{"public.temp", "public.cache"},
		},
		{
			name:              "both include and skip",
			includeTablesStr:  "users,posts",
			skipTablesStr:     "temp",
			schemas:           []string{"public"},
			wantIncludeTables: []string{"public.users", "public.posts"},
			wantSkipTables:    []string{"public.temp"},
		},
		{
			name:              "whitespace handling",
			includeTablesStr:  " users , posts ",
			skipTablesStr:     " temp , cache ",
			schemas:           []string{"public"},
			wantIncludeTables: []string{"public.users", "public.posts"},
			wantSkipTables:    []string{"public.temp", "public.cache"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInclude, gotSkip := parseTableFilters(tt.includeTablesStr, tt.skipTablesStr, tt.schemas)
			if !reflect.DeepEqual(gotInclude, tt.wantIncludeTables) {
				t.Errorf("parseTableFilters() include = %v, want %v", gotInclude, tt.wantIncludeTables)
			}
			if !reflect.DeepEqual(gotSkip, tt.wantSkipTables) {
				t.Errorf("parseTableFilters() skip = %v, want %v", gotSkip, tt.wantSkipTables)
			}
		})
	}
}

func TestFilterSchemas(t *testing.T) {
	tests := []struct {
		name           string
		schemas        []string
		excludeSchemas []string
		want           []string
		wantErr        bool
	}{
		{
			name:           "no exclusions",
			schemas:        []string{"public", "app"},
			excludeSchemas: []string{},
			want:           []string{"public", "app"},
		},
		{
			name:           "exclude one schema",
			schemas:        []string{"public", "app", "temp"},
			excludeSchemas: []string{"temp"},
			want:           []string{"public", "app"},
		},
		{
			name:           "exclude multiple schemas",
			schemas:        []string{"public", "app", "auth", "storage"},
			excludeSchemas: []string{"auth", "storage"},
			want:           []string{"public", "app"},
		},
		{
			name:           "exclude all schemas",
			schemas:        []string{"public"},
			excludeSchemas: []string{"public"},
			want:           nil,
			wantErr:        true,
		},
		{
			name:           "exclude non-existent schema",
			schemas:        []string{"public", "app"},
			excludeSchemas: []string{"nonexistent"},
			want:           []string{"public", "app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterSchemas(tt.schemas, tt.excludeSchemas)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterSchemas() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterSchemas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *postgres.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &postgres.Config{
				Host: "db.example.com",
				Port: 5432,
			},
			wantErr: false,
		},
		{
			name: "pooler hostname",
			cfg: &postgres.Config{
				Host: "pooler-db.example.com",
				Port: 5432,
			},
			wantErr: true,
		},
		{
			name: "pooler port",
			cfg: &postgres.Config{
				Host: "db.example.com",
				Port: 6543,
			},
			wantErr: true,
		},
		{
			name: "supabase pooler",
			cfg: &postgres.Config{
				Host: "db.pooler.supabase.com",
				Port: 6543,
			},
			wantErr: true,
		},
		{
			name: "ipv6 address",
			cfg: &postgres.Config{
				Host: "2001:db8::1",
				Port: 5432,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSourceConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSourceConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
