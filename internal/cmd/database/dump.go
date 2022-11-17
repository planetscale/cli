package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/dumper"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/proxyutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/sql-proxy/proxy"

	_ "github.com/go-sql-driver/mysql"

	"github.com/spf13/cobra"
)

type dumpFlags struct {
	localAddr string
	tables    string
	wheres    string
	output    string
	threads   int
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

	cmd.PersistentFlags().StringVar(&f.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
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

	client, err := ch.Client()
	if err != nil {
		return err
	}

	const localProxyAddr = "127.0.0.1"
	localAddr := localProxyAddr + ":0"
	if flags.localAddr != "" {
		localAddr = flags.localAddr
	}

	proxyOpts := proxy.Options{
		CertSource: proxyutil.NewRemoteCertSource(client, cmdutil.AdministratorRole),
		LocalAddr:  localAddr,
		Instance:   fmt.Sprintf("%s/%s/%s", ch.Config.Organization, database, branch),
		Logger:     cmdutil.NewZapLogger(ch.Debug()),
	}

	p, err := proxy.NewClient(proxyOpts)
	if err != nil {
		return fmt.Errorf("couldn't create proxy client: %s", err)
	}

	go func() {
		err := p.Run(ctx)
		if err != nil {
			ch.Printer.Println("proxy error: ", err)
		}
	}()

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

	addr, err := p.LocalAddr()
	if err != nil {
		return err
	}

	dbName, err := getDatabaseName(database, addr.String())
	if err != nil {
		return err
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, fmt.Sprintf("pscale_dump_%s_%s", database, branch))

	if flags.output != "" {
		dir = flags.output
	}

	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("backup directory already exists: %s", dir)
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	cfg := dumper.NewDefaultConfig()
	cfg.Threads = flags.threads
	cfg.User = "root"
	// NOTE(fatih): the password is a placeholder, replace once we get rid of the proxy
	cfg.Password = "root"
	cfg.Address = addr.String()
	cfg.Database = dbName
	cfg.Debug = ch.Debug()
	cfg.StmtSize = 1000000
	cfg.IntervalMs = 10 * 1000
	cfg.ChunksizeInMB = 128
	cfg.SessionVars = "set workload=olap;"
	cfg.Outdir = dir

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
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/", "root", "", addr)
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

	var hasDatabaseName = map[string]bool{
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
