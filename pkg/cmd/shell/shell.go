package shell

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/pkg/promptutil"
	"github.com/planetscale/cli/pkg/proxyutil"
	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"

	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	localProxyPort = "3307"
	localProxyAddr = "127.0.0.1"
)

func ShellCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
		verbose    bool
	}

	cmd := &cobra.Command{
		Use:   "shell [database] [branch]",
		Short: "Open a shell instance to the given database and branch",
		Example: `The shell subcommand opens a secure MySQL shell instance to your database.

It uses the "mysqli" 
By default, if no branch names are given and there is only one branch, it
automatically opens a shell to that branch:

  pscale shell mydatabase
 
If there are multiple branches for the given database, you'll be prompted to
choose one. To open a shell instance to a specific branch, pass the branch as a
second argument:

  pscale shell mydatabase mybranch`,
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

			if !flags.verbose {
				proxyOpts.Logger = zap.NewNop()
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
					fmt.Println("proxy error: ", err)
				}
			}()

			status, err := client.DatabaseBranches.GetStatus(ctx, &ps.GetDatabaseBranchStatusRequest{
				Organization: cfg.Organization,
				Database:     database,
				Branch:       branch,
			})

			mysqlArgs := []string{
				"-u", status.User,
				fmt.Sprintf("-p%s", status.Password),
				"-h", localProxyAddr,
				"-P", localProxyPort,
			}

			m := &mysql{}
			err = m.Run(ctx, mysqlArgs...)
			return err

		},
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&flags.localAddr, "local-addr",
		localProxyAddr+":"+localProxyPort, "Local address to bind and listen for connections")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API.")
	cmd.PersistentFlags().BoolVar(&flags.verbose, "v", false, "enable verbose mode")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}

type mysql struct {
	Dir string
}

// Run runs the `mysql` client with the given arguments
func (m *mysql) Run(ctx context.Context, args ...string) error {
	c := exec.CommandContext(ctx, "mysql", args...)
	if m.Dir != "" {
		c.Dir = m.Dir
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	err := c.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = c.Wait()
	return err

}
