package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/dumper"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/proxyutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/sql-proxy/proxy"

	"github.com/spf13/cobra"
)

type restoreFlags struct {
	localAddr string
	dir       string
}

// RestoreCmd encapsulates the commands for restore a database
func RestoreCmd(ch *cmdutil.Helper) *cobra.Command {
	f := &restoreFlags{}
	cmd := &cobra.Command{
		Use:   "restore-dump <database> <branch> [options]",
		Short: "Restore your database from a local dump directory",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE:  func(cmd *cobra.Command, args []string) error { return restore(ch, cmd, f, args) },
	}

	cmd.PersistentFlags().StringVar(&f.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&f.dir, "dir", "",
		"Directory that contains the files to be used for restoration (required)")

	return cmd
}

func restore(ch *cmdutil.Helper, cmd *cobra.Command, flags *restoreFlags, args []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	database := args[0]
	branch := args[1]

	if flags.dir == "" {
		return errors.New("--dir flag is missing, it's needed to restore the database")
	}

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

	if !status.Ready {
		return errors.New("database branch is not ready yet, please try again in a few minutes")
	}

	addr, err := p.LocalAddr()
	if err != nil {
		return err
	}

	cfg := dumper.NewDefaultConfig()
	cfg.User = "root"
	cfg.Address = addr.String()
	cfg.Debug = ch.Debug()
	cfg.IntervalMs = 10 * 1000
	cfg.Outdir = flags.dir

	loader, err := dumper.NewLoader(cfg)
	if err != nil {
		return err
	}

	ch.Printer.Printf("Starting to restore database %s from folder %s\n",
		printer.BoldBlue(database), printer.BoldBlue(flags.dir))

	end := ch.Printer.PrintProgress("Restoring database ...")
	defer end()

	start := time.Now()
	err = loader.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to restore database: %s", err)
	}

	end()
	ch.Printer.Printf("Restoring is finished! (elapsed time: %s)\n", time.Since(start))
	return nil
}
