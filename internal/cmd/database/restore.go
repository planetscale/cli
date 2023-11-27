package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/dumper"
	"github.com/planetscale/cli/internal/passwordutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/proxyutil"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

type restoreFlags struct {
	localAddr  string
	remoteAddr string
	dir        string
	overwrite  bool
	threads    int
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
	cmd.PersistentFlags().StringVar(&f.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API. (format: `hostname:port`)")
	cmd.PersistentFlags().StringVar(&f.dir, "dir", "",
		"Directory containing the files to be used for the restore (required)")
	cmd.PersistentFlags().BoolVar(&f.overwrite, "overwrite-tables", false, "If true, will attempt to DROP TABLE before restoring.")

	cmd.PersistentFlags().IntVar(&f.threads, "threads", 1, "Number of concurrent threads to use to restore the database.")
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
		Name:         passwordutil.GenerateName("pscale-cli-restore"),
		TTL:          6 * time.Hour, // TODO: use shorter TTL, but implement refreshing
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

	addr := l.Addr()

	cfg := dumper.NewDefaultConfig()
	cfg.Threads = flags.threads
	// NOTE(mattrobenolt): credentials are needed even though they aren't used,
	// otherwise, dumper will complain.
	cfg.User = "nobody"
	cfg.Password = "nobody"
	cfg.Address = addr.String()
	cfg.Debug = ch.Debug()
	cfg.IntervalMs = 10 * 1000
	cfg.Outdir = flags.dir
	cfg.OverwriteTables = flags.overwrite

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
	ch.Printer.Printf("Restore is finished! (elapsed time: %s)\n", time.Since(start))
	return nil
}
