package database

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/proxyutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"

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
		Use:    "restore <database> <branch> [options]",
		Short:  "Restore your database",
		Args:   cmdutil.RequiredArgs("database", "branch"),
		Hidden: true,
		RunE:   func(cmd *cobra.Command, args []string) error { return restore(ch, cmd, f, args) },
	}

	cmd.PersistentFlags().StringVar(&f.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&f.dir, "dir", "",
		"Directory that contains the files to be used for restoration (required)")

	return cmd
}

func restore(ch *cmdutil.Helper, cmd *cobra.Command, flags *restoreFlags, args []string) error {
	return errors.New("the restore functionality is not implemented yet")

	ctx := context.Background() //nolint: govet
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

	// address to talk for dumping
	_, err = p.LocalAddr()
	if err != nil {
		return err
	}

	ch.Printer.Printf("Starting to restore database %s from folder %s\n",
		printer.BoldBlue(database), printer.BoldBlue(flags.dir))

	end := ch.Printer.PrintProgress("Restoring database ...")
	defer end()

	start := time.Now()

	// start the restoring here ...

	end()
	ch.Printer.Printf("Restoring is finished! (elapsed time: %s)\n", time.Since(start))
	return nil
}
