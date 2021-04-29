package dumper

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

// Config describes the settings to dump from a database.
type Config struct {
	User                 string
	Password             string
	Address              string
	ToUser               string
	ToPassword           string
	ToAddress            string
	ToDatabase           string
	ToEngine             string
	Database             string
	DatabaseRegexp       string
	DatabaseInvertRegexp bool
	Table                string
	Outdir               string
	SessionVars          string
	Format               string
	Threads              int
	ChunksizeInMB        int
	StmtSize             int
	Allbytes             uint64
	Allrows              uint64
	OverwriteTables      bool
	Wheres               map[string]string
	Selects              map[string]map[string]string
	Filters              map[string]map[string]string

	// Interval in millisecond.
	IntervalMs int
}

func ParseDumperConfig(file string) (*Config, error) {
	args := &Config{
		Wheres: make(map[string]string),
	}

	cfg, err := ini.Load(file)
	if err != nil {
		return nil, err
	}

	host := cfg.Section("mysql").Key("host").String()
	if host == "" {
		return nil, errors.New("empty host")
	}
	port, err := cfg.Section("mysql").Key("port").Int()
	if port == 0 || err != nil {
		return nil, errors.New("invalid port")
	}

	user := cfg.Section("mysql").Key("user").String()
	if user == "" {
		return nil, errors.New("empty user")
	}

	password := cfg.Section("mysql").Key("password").String()
	database := cfg.Section("mysql").Key("database").String()
	outdir := cfg.Section("mysql").Key("outdir").String()
	if outdir == "" {
		return nil, errors.New("empty outdir")
	}
	sessionVars := cfg.Section("mysql").Key("vars").String()
	format := cfg.Section("mysql").Key("format").String()
	chunksizemb, err := cfg.Section("mysql").Key("chunksize").Int()
	if err != nil {
		return nil, fmt.Errorf("pasre mysql.chunksize failed")
	}
	table := cfg.Section("mysql").Key("table").String()

	// Options
	if err := LoadOptions(cfg, "where", args.Wheres); err != nil {
		return nil, err
	}

	selects := cfg.Section("select").Keys()
	for _, tblcol := range selects {
		var table, column string
		split := strings.Split(tblcol.Name(), ".")
		table = split[0]
		column = split[1]

		if args.Selects == nil {
			args.Selects = make(map[string]map[string]string)
		}
		if args.Selects[table] == nil {
			args.Selects[table] = make(map[string]string)
		}
		args.Selects[table][column] = tblcol.String()
	}

	database_regexp := cfg.Section("database").Key("regexp").String()
	database_invert_regexp, err := cfg.Section("database").Key("invert_regexp").Bool()
	if err != nil {
		database_invert_regexp = false
	}

	filters := cfg.Section("filter").Keys()
	for _, tblcol := range filters {
		var table, column string
		split := strings.Split(tblcol.Name(), ".")
		table = split[0]
		column = split[1]

		if args.Filters == nil {
			args.Filters = make(map[string]map[string]string)
		}
		if args.Filters[table] == nil {
			args.Filters[table] = make(map[string]string)
		}
		args.Filters[table][column] = tblcol.String()
	}

	args.Address = fmt.Sprintf("%s:%d", host, port)
	args.User = user
	args.Password = password
	args.Database = database
	args.DatabaseRegexp = database_regexp
	args.DatabaseInvertRegexp = database_invert_regexp
	args.Table = table
	args.Outdir = outdir
	args.ChunksizeInMB = chunksizemb
	args.SessionVars = sessionVars
	args.Format = format
	args.Threads = 16
	args.StmtSize = 1000000
	args.IntervalMs = 10 * 1000
	return args, nil
}

func LoadOptions(cfg *ini.File, section string, optMap map[string]string) error {
	opts := cfg.Section(section).Keys()

	fmt.Printf("sec=%s keys=%+v\n", section, opts)

	for _, key := range opts {
		optMap[key.Name()] = key.String()
		fmt.Printf("sec=%s %s=%s\n", section, key.Name(), key.String())
	}
	return nil
}
