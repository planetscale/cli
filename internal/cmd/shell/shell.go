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
	"path/filepath"
	"syscall"

	"github.com/mitchellh/go-homedir"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/promptutil"
	"github.com/planetscale/cli/internal/proxyutil"

	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"

	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ShellCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
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
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]

			if !printer.IsTTY || ch.Printer.Format() != printer.Human {
				return errors.New("pscale shell only works in interactive mode")
			}

			_, err := exec.LookPath("mysql")
			if err != nil {
				return fmt.Errorf("couldn't find the 'mysql' CLI")
			}

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			var branch string
			if len(args) == 2 {
				branch = args[1]
			}

			if branch == "" {
				branch, err = promptutil.GetBranch(ctx, client, ch.Config.Organization, database)
				if err != nil {
					return err
				}
			}

			// check whether database and branch exist
			_, err = client.DatabaseBranches.Get(ctx, &ps.GetDatabaseBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s and branch %s does not exist in organization %s\n",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
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
				return errors.New("database branch is not ready yet")
			}

			tmpFile, err := createLoginFile(status.Credentials.User, status.Credentials.Password)
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
				"-s",
				"-t", // the -s (silent) flag disables tabular output, re-enable it.
				"-h", host,
				"-P", port,
			}

			historyFile, err := historyFilePath(ch.Config.Organization, database, branch)
			if err != nil {
				return err
			}

			styledBranch := formatMySQLBranch(database, branch)
			m := &mysql{}
			err = m.Run(ctx, historyFile, styledBranch, mysqlArgs...)
			return err

		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&flags.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API.")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}

func historyFilePath(org, db, branch string) (string, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	historyDir := filepath.Join(dir, ".pscale", "history")

	_, err = os.Stat(historyDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(historyDir, 0771)
		if err != nil {
			return "", err
		}
	}

	historyFilename := fmt.Sprintf("%s.%s.%s", org, db, branch)
	historyFile := filepath.Join(historyDir, historyFilename)

	return historyFile, nil
}

func formatMySQLBranch(database, branch string) string {
	branchStyled := printer.BoldBlue(branch)
	if branch == "main" {
		branchStyled = printer.BoldRed(branch)
	}

	return fmt.Sprintf("%s/%s> ", printer.Bold(database), branchStyled)
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
func (m *mysql) Run(ctx context.Context, historyFile string, styledBranch string, args ...string) error {
	c := exec.CommandContext(ctx, "mysql", args...)
	if m.Dir != "" {
		c.Dir = m.Dir
	}

	c.Env = append(os.Environ(),
		fmt.Sprintf("MYSQL_PS1=%s", styledBranch),
		fmt.Sprintf("MYSQL_HISTFILE=%s", historyFile),
	)

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
