package postgres

import (
	"testing"
)

func TestParseConnectionURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    *Config
		wantErr bool
	}{
		{
			name: "basic uri",
			uri:  "postgresql://user:pass@localhost:5432/mydb",
			want: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "mydb",
				SSLMode:  "require",
				Options:  make(map[string]string),
			},
		},
		{
			name: "uri with sslmode",
			uri:  "postgresql://user:pass@localhost:5432/mydb?sslmode=disable",
			want: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "mydb",
				SSLMode:  "disable",
				Options:  make(map[string]string),
			},
		},
		{
			name: "uri without password",
			uri:  "postgresql://user@localhost:5432/mydb",
			want: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "",
				Database: "mydb",
				SSLMode:  "require",
				Options:  make(map[string]string),
			},
		},
		{
			name: "key-value format",
			uri:  "host=localhost port=5432 dbname=mydb",
			want: &Config{
				Host:     "localhost",
				Port:     5432,
				Database: "mydb",
				SSLMode:  "require",
				Options:  make(map[string]string),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConnectionURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConnectionURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Host != tt.want.Host {
				t.Errorf("Host = %v, want %v", got.Host, tt.want.Host)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %v, want %v", got.Port, tt.want.Port)
			}
			if got.User != tt.want.User {
				t.Errorf("User = %v, want %v", got.User, tt.want.User)
			}
			if got.Password != tt.want.Password {
				t.Errorf("Password = %v, want %v", got.Password, tt.want.Password)
			}
			if got.Database != tt.want.Database {
				t.Errorf("Database = %v, want %v", got.Database, tt.want.Database)
			}
			if got.SSLMode != tt.want.SSLMode {
				t.Errorf("SSLMode = %v, want %v", got.SSLMode, tt.want.SSLMode)
			}
		})
	}
}

func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "basic config",
			cfg: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "mydb",
				SSLMode:  "require",
				Options:  make(map[string]string),
			},
			want: "host=localhost port=5432 user=user password=pass dbname=mydb sslmode=require",
		},
		{
			name: "config without password",
			cfg: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Database: "mydb",
				SSLMode:  "disable",
				Options:  make(map[string]string),
			},
			want: "host=localhost port=5432 user=user dbname=mydb sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildConnectionString(tt.cfg)
			if got != tt.want {
				t.Errorf("BuildConnectionString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedactPassword(t *testing.T) {
	tests := []struct {
		name   string
		connStr string
		want   string
	}{
		{
			name:   "with password",
			connStr: "host=localhost port=5432 user=user password=secret dbname=mydb",
			want:   "host=localhost port=5432 user=user password=**** dbname=mydb",
		},
		{
			name:   "without password",
			connStr: "host=localhost port=5432 user=user dbname=mydb",
			want:   "host=localhost port=5432 user=user dbname=mydb",
		},
		{
			name:   "empty string",
			connStr: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactPassword(tt.connStr)
			if got != tt.want {
				t.Errorf("RedactPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "simple identifier",
			id:   "mytable",
			want: `"mytable"`,
		},
		{
			name: "identifier with quotes",
			id:   `table"name`,
			want: `"table""name"`,
		},
		{
			name: "identifier with multiple quotes",
			id:   `my"table"name`,
			want: `"my""table""name"`,
		},
		{
			name: "empty string",
			id:   "",
			want: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuoteIdentifier(tt.id)
			if got != tt.want {
				t.Errorf("QuoteIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}
