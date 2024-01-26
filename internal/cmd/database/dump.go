package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/dumper"
	"github.com/planetscale/cli/internal/passwordutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/proxyutil"
	ps "github.com/planetscale/planetscale-go/planetscale"

	_ "github.com/go-sql-driver/mysql"

	"github.com/spf13/cobra"
)

type dumpFlags struct {
	localAddr  string
	remoteAddr string
	keyspace   string
	replica    bool
	tables     string
	wheres     string
	output     string
	threads    int
}

// DumpCmd encapsulates the commands for dumping a database
func DumpCmd(ch *cmdutil.Helper) *cobra.Command {
	f := &dumpFlags{}
	cmd := &cobra.Command{
		Use:   "dump <database> <branch> [options]",
		Short: "Backup and dump your database",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE:  func(cmd *cobra.Command, args []string) error { return dump(ch, cmd, f, args) },
	}

	cmd.PersistentFlags().StringVar(&f.keyspace, "keyspace",
		"", "Optionally target a specific keyspace to be dumped. Useful for sharded databases.")
	cmd.PersistentFlags().StringVar(&f.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&f.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API. (format: `hostname:port`)")
	cmd.PersistentFlags().BoolVar(&f.replica, "replica", false, "Dump from a replica (if available; will fail if not).")
	cmd.PersistentFlags().StringVar(&f.tables, "tables", "",
		"Comma separated string of tables to dump. By default all tables are dumped.")
	cmd.PersistentFlags().StringVar(&f.wheres, "wheres", "",
		"Comma separated string of WHERE clauses to filter the tables to dump. Only used when you specify tables to dump. Default is not to filter dumped tables.")
	cmd.PersistentFlags().StringVar(&f.output, "output", "",
		"Output directory of the dump. By default the dump is saved to a folder in the current directory.")
	cmd.PersistentFlags().IntVar(&f.threads, "threads", 16, "Number of concurrent threads to use to dump the database.")

	return cmd
}

func dump(ch *cmdutil.Helper, cmd *cobra.Command, flags *dumpFlags, args []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	database := args[0]
	branch := args[1]
	keyspace := flags.keyspace

	if keyspace == "" {
		keyspace = database
	}

	client, err := ch.Client()
	if err != nil {
		return err
	}

	db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
		Organization: ch.Config.Organization,
		Database:     database,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return fmt.Errorf("database %s does not exist in organization: %s",
				printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
		default:
			return cmdutil.HandleError(err)
		}
	}

	if db.State == ps.DatabaseSleeping {
		return fmt.Errorf("database %s is sleeping, please wake the database and retry this command", printer.BoldBlue(database))
	}

	if db.State == ps.DatabaseAwakening {
		return fmt.Errorf("database %s is waking from sleep, please wait until it's ready and retry this command", printer.BoldBlue(database))
	}

	dbBranch, err := client.DatabaseBranches.Get(ctx, &ps.GetDatabaseBranchRequest{
		Organization: ch.Config.Organization,
		Database:     database,
		Branch:       branch,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
				printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
		default:
			return cmdutil.HandleError(err)
		}
	}

	if !dbBranch.Ready {
		return errors.New("database branch is not ready yet, please try again in a few minutes")
	}

	pw, err := passwordutil.New(ctx, client, passwordutil.Options{
		Organization: ch.Config.Organization,
		Database:     database,
		Branch:       branch,
		Role:         cmdutil.AdministratorRole,
		Name:         passwordutil.GenerateName("pscale-cli-dump"),
		TTL:          5 * time.Minute,
	})
	if err != nil {
		return cmdutil.HandleError(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := pw.Cleanup(ctx); err != nil {
			ch.Printer.Println("failed to delete credentials: ", err)
		}
	}()

	localAddr := "127.0.0.1:0"
	if flags.localAddr != "" {
		localAddr = flags.localAddr
	}

	remoteAddr := flags.remoteAddr
	if remoteAddr == "" {
		remoteAddr = pw.Password.Hostname
	}

	proxy := proxyutil.New(proxyutil.Config{
		Logger:       cmdutil.NewZapLogger(ch.Debug()),
		UpstreamAddr: remoteAddr,
		Username:     pw.Password.Username,
		Password:     pw.Password.PlainText,
	})
	defer proxy.Close()

	l, err := net.Listen("tcp", localAddr)
	if err != nil {
		return cmdutil.HandleError(err)
	}
	defer l.Close()

	go func() {
		if err := proxy.Serve(l); err != nil {
			ch.Printer.Println("proxy error: ", err)
		}
	}()

	go func() {
		if err := pw.Renew(ctx); err != nil {
			ch.Printer.Println("proxy error: ", err)
		}
	}()

	addr := l.Addr()

	dbName, err := getDatabaseName(keyspace, addr.String())
	if err != nil {
		return err
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	if dbName == database {
		dir = filepath.Join(dir, fmt.Sprintf("pscale_dump_%s_%s_%s", database, branch, timestamp))
	} else {
		dir = filepath.Join(dir, fmt.Sprintf("pscale_dump_%s_%s_%s_%s", database, branch, dbName, timestamp))
	}

	if flags.output != "" {
		dir = flags.output
	}

	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("backup directory already exists: %s", dir)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	cfg := dumper.NewDefaultConfig()
	cfg.Threads = flags.threads
	// NOTE(mattrobenolt): credentials are needed even though they aren't used,
	// otherwise, dumper will complain.
	cfg.User = "nobody"
	cfg.Password = "nobody"
	cfg.Address = addr.String()
	cfg.Database = dbName
	cfg.Debug = ch.Debug()
	cfg.StmtSize = 1000000
	cfg.IntervalMs = 10 * 1000
	cfg.ChunksizeInMB = 128
	cfg.SessionVars = "set workload=olap;"
	cfg.Outdir = dir

	if flags.replica {
		cfg.UseReplica = true
	}

	if flags.tables != "" {
		cfg.Table = flags.tables
		if flags.wheres != "" {
			m := make(map[string]string)
			tables := strings.Split(flags.tables, ",")
			wheres := strings.Split(flags.wheres, ",")
			for i := range wheres {
				m[tables[i]] = wheres[i]
			}
			cfg.Wheres = m
		}
	}

	d, err := dumper.NewDumper(cfg)
	if err != nil {
		return err
	}

	if flags.tables == "" {
		ch.Printer.Printf("Starting to dump all tables from database %s to folder %s\n",
			printer.BoldBlue(database), printer.Bold(dir))
	} else {
		ch.Printer.Printf("Starting to dump tables '%s' from database %s to folder %s\n",
			printer.BoldRed(flags.tables), printer.BoldBlue(database), printer.BoldBlue(dir))
	}

	end := ch.Printer.PrintProgress("Dumping tables ...")
	defer end()

	start := time.Now()
	err = d.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to dump database: %s", err)
	}

	end()
	ch.Printer.Printf("Dumping is finished! (elapsed time: %s)\n", time.Since(start))
	return nil
}

func getDatabaseName(name, addr string) (string, error) {
	dsn := fmt.Sprintf("tcp(%s)/", addr)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", err
	}
	defer db.Close()

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var dbs []string
	for rows.Next() {
		var db string
		if err := rows.Scan(&db); err != nil {
			return "", err
		}

		if name == db {
			return db, nil
		}

		dbs = append(dbs, db)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("failed getting database names: %s", err)
	}

	hasDatabaseName := map[string]bool{
		"onboarding-demo": true,
	}

	for _, v := range dbs {
		if hasDatabaseName[v] {
			return v, nil
		}
	}

	// this means we didn't find a match.
	return "", errors.New("could not find a valid database name for this database")
}
