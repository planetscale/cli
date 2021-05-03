package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/dumper"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/proxyutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"

	"github.com/spf13/cobra"
)

type dumpFlags struct {
	localAddr string
	tables    string
	output    string
}

// DumpCmd encapsulates the commands for dumping a databse
func DumpCmd(ch *cmdutil.Helper) *cobra.Command {
	f := &dumpFlags{}
	cmd := &cobra.Command{
		Use:   "dump <database> <branch> [options]",
		Short: "Backup and dump your database",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE:  func(cmd *cobra.Command, args []string) error { return run(ch, cmd, f, args) },
	}

	cmd.PersistentFlags().StringVar(&f.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&f.tables, "tables", "",
		"Comma separated string of tables to dump. By default all tables are dumped.")
	cmd.PersistentFlags().StringVar(&f.output, "output", "",
		"Output director of the dump. By default the dump is stored to a folder in the current directory.")

	return cmd
}

func run(ch *cmdutil.Helper, cmd *cobra.Command, flags *dumpFlags, args []string) error {
	ctx := context.Background()
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
		CertSource: proxyutil.NewRemoteCertSource(client),
		LocalAddr:  localAddr,
		Instance:   fmt.Sprintf("%s/%s/%s", ch.Config.Organization, database, branch),
		Logger:     cmdutil.NewZapLogger(ch.Debug()),
	}

	p, err := proxy.NewClient(proxyOpts)
	if err != nil {
		return fmt.Errorf("couldn't create proxy client: %s", err)
	}

	// TODO(fatih): replace with signal.NotifyContext once Go 1.16 is released
	// https://go-review.googlesource.com/c/go/+/219640
	ctx = sigutil.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err := p.Run(ctx)
		if err != nil {
			ch.Printer.Println("proxy error: ", err)
		}
	}()

	status, err := client.DatabaseBranches.GetStatus(ctx, &ps.GetDatabaseBranchStatusRequest{
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

	if status.Credentials.User == "" {
		return errors.New("database branch is not ready yet, please try again in a few minutes.")
	}

	addr, err := p.LocalAddr()
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
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	dumperCfg := dumper.NewDefaultConfig()
	dumperCfg.User = status.Credentials.User
	dumperCfg.Password = status.Credentials.Password
	dumperCfg.Address = addr.String()
	dumperCfg.Database = database
	dumperCfg.Debug = ch.Debug()

	if flags.tables != "" {
		dumperCfg.Table = flags.tables
	}
	dumperCfg.Database = database
	dumperCfg.Outdir = dir

	d, err := dumper.NewDumper(dumperCfg)
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
