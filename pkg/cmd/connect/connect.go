package connect

import (
	"context"
	"fmt"
	"syscall"

	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/pkg/promptutil"
	"github.com/planetscale/cli/pkg/proxyutil"
	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func ConnectCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
		debug      bool
	}

	cmd := &cobra.Command{
		Use:   "connect [database] [branch]",
		Short: "Create a secure connection to the given database and branch",
		Example: `The connect subcommand establish a secure connection between your host and remote psdb. 

By default, if no branch names are given and there is only one branch, it
automatically connects to that branch:

  pscale connect mydatabase
 
If there are multiple branches for the given database, you'll be prompted to
choose one. To connect to a specific branch, pass the branch as a second
argument:

  pscale connect mydatabase mybranch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) < 1 {
				return cmd.Usage()
			}

			database := args[0]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			var branch string
			if len(args) == 2 {
				branch = args[1]
			}

			if branch == "" {
				branch, err = promptutil.GetBranch(ctx, client, cfg.Organization, database)
				if err != nil {
					return err
				}
			}

			proxyOpts := proxy.Options{
				CertSource: proxyutil.NewRemoteCertSource(client),
				LocalAddr:  flags.localAddr,
				RemoteAddr: flags.remoteAddr,
				Instance:   fmt.Sprintf("%s/%s/%s", cfg.Organization, database, branch),
			}

			if !flags.debug {
				proxyOpts.Logger = zap.NewNop()
			}

			fmt.Printf("Secure connection to databases %s and branch %s is established!.\n\nLocal address to connect your application: %s (press ctrl-c to quit)",
				cmdutil.BoldBlue(database),
				cmdutil.BoldBlue(branch),
				cmdutil.BoldBlue(flags.localAddr))

			p, err := proxy.NewClient(proxyOpts)
			if err != nil {
				return fmt.Errorf("couldn't create proxy client: %s", err)
			}

			// TODO(fatih): replace with signal.NotifyContext once Go 1.16 is released
			// https://go-review.googlesource.com/c/go/+/219640
			ctx = sigutil.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
			return p.Run(ctx)
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&flags.localAddr, "local-addr", "127.0.0.1:3307",
		"Local address to bind and listen for connections")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API.")
	cmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "enable debug mode")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}
