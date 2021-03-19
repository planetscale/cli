package shell

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/promptutil"
	"github.com/planetscale/cli/internal/proxyutil"

	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"

	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func ShellCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
		debug      bool
	}

	cmd := &cobra.Command{
		Use: "shell [database] [branch]",
		// we only require database, because we deduct branch automatically
		Args:  cmdutil.RequiredArgs("database"),
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
			database := args[0]

			_, err := exec.LookPath("mysql")
			if err != nil {
				return fmt.Errorf("couldn't find the 'mysql' CLI")
			}

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

			const localProxyAddr = "127.0.0.1"
			localAddr := localProxyAddr + ":0"
			if flags.localAddr != "" {
				localAddr = flags.localAddr
			}

			proxyOpts := proxy.Options{
				CertSource: proxyutil.NewRemoteCertSource(client),
				LocalAddr:  localAddr,
				RemoteAddr: flags.remoteAddr,
				Instance:   fmt.Sprintf("%s/%s/%s", cfg.Organization, database, branch),
			}

			if !flags.debug {
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
			if err != nil {
				return err
			}

			if status.User == "" {
				return errors.New("database branch is not ready yet")
			}

			tmpFile, err := createLoginFile(status.User, status.Password)
			if tmpFile != "" {
				defer os.Remove(tmpFile)
			}
			if err != nil {
				return err
			}

			addr, err := p.LocalAddr()
			if err != nil {
				return err
			}

			host, port, err := net.SplitHostPort(addr.String())
			if err != nil {
				return err
			}

			mysqlArgs := []string{
				fmt.Sprintf("--defaults-extra-file=%s", tmpFile),
				"-h", host,
				"-P", port,
			}

			m := &mysql{}
			err = m.Run(ctx, mysqlArgs...)
			return err

		},
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&flags.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API.")
	cmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "enable debug mode")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}

// createLoginFile creates a temporary file to store the username and password, so we don't have to
// pass them as `mysql` command-line arguments.
func createLoginFile(username, password string) (string, error) {
	// ioutil.TempFile defaults to creating the file in the OS temporary directory with 0600 permissions
	tmpFile, err := ioutil.TempFile("", "pscale-*")
	if err != nil {
		fmt.Println("could not create temporary file: ", err)
		return "", err
	}
	fmt.Fprintln(tmpFile, "[client]")
	fmt.Fprintf(tmpFile, "user=%s\n", username)
	fmt.Fprintf(tmpFile, "password=%s\n", password)
	_ = tmpFile.Close()
	return tmpFile.Name(), nil
}

type mysql struct {
	Dir string
}

// Run runs the `mysql` client with the given arguments.
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
