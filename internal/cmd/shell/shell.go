package shell

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/promptutil"
	"github.com/planetscale/cli/internal/proxyutil"

	"github.com/planetscale/sql-proxy/proxy"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	exec "golang.org/x/sys/execabs"
)

func ShellCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
		role       string
	}

	cmd := &cobra.Command{
		Use: "shell [database] [branch]",
		// we only require database, because we deduct branch automatically
		Args:  cmdutil.RequiredArgs("database"),
		Short: "Open a MySQL shell instance to a database and branch",
		Example: `The shell subcommand opens a secure MySQL shell instance to your database.

It uses the MySQL command-line client ("mysql"), which needs to be installed.
By default, if no branch names are given and there is only one branch, it
automatically opens a shell to that branch:

  pscale shell mydatabase

If there are multiple branches for the given database, you'll be prompted to
choose one. To open a shell instance to a specific branch, pass the branch as a
second argument:

  pscale shell mydatabase mybranch`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			database := args[0]

			if !printer.IsTTY || ch.Printer.Format() != printer.Human {
				if _, exists := os.LookupEnv("PSCALE_ALLOW_NONINTERACTIVE_SHELL"); !exists {
					return errors.New("pscale shell only works in interactive mode")
				}
			}

			mysqlPath, err := cmdutil.MySQLClientPath()
			if err != nil {
				return err
			}

			client, err := ch.Client()
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

			role := cmdutil.AdministratorRole
			if flags.role != "" {
				role, err = cmdutil.RoleFromString(flags.role)
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
					return fmt.Errorf("database %s and branch %s does not exist in organization %s",
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
				CertSource: proxyutil.NewRemoteCertSource(client, role),
				LocalAddr:  localAddr,
				RemoteAddr: flags.remoteAddr,
				Instance:   fmt.Sprintf("%s/%s/%s", ch.Config.Organization, database, branch),
				Logger:     cmdutil.NewZapLogger(ch.Debug()),
			}

			proxyAddr := make(chan string, 1)
			proxyError := make(chan error, 1)

			go func() {
				proxyError <- runProxy(ctx, ch, proxyOpts, proxyAddr)
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
				return errors.New("database branch is not ready yet")
			}

			var addr string
			select {
			case err := <-proxyError:
				return err
			case addr = <-proxyAddr:
			case <-time.After(time.Second * 10):
				return errors.New("proxy timeout retrieving the certs")
			}

			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return err
			}

			mysqlArgs := []string{
				"-u",
				"root",
				"-c", // allow comments to pass to the server
				"-s",
				"-t", // the -s (silent) flag disables tabular output, re-enable it.
				"-h", host,
				"-P", port,
			}

			historyFile, err := historyFilePath(ch.Config.Organization, database, branch)
			if err != nil {
				return err
			}

			styledBranch := formatMySQLBranch(database, dbBranch)

			m := &mysql{
				mysqlPath:    mysqlPath,
				historyFile:  historyFile,
				styledBranch: styledBranch,
				debug:        ch.Debug(),
				printer:      ch.Printer,
			}

			err = m.Run(ctx, mysqlArgs...)
			return err

		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&flags.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API.")
	cmd.PersistentFlags().StringVar(&flags.role, "role",
		"admin", "Role defines the access level, allowed values are : reader, writer, readwriter, admin. By default it is admin.")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck
	cmd.PersistentFlags().MarkHidden("role")
	return cmd
}

// runProxy runs the sql-proxy with the given options.
func runProxy(ctx context.Context, ch *cmdutil.Helper, proxyOpts proxy.Options, ready chan string) error {
	p, err := proxy.NewClient(proxyOpts)
	if err != nil {
		return fmt.Errorf("couldn't create proxy client: %s", err)
	}

	go func(ready chan string) {
		// this is blocking and will only return once p.Run() below is
		// invoked
		addr, err := p.LocalAddr()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed getting local addr: %s\n", err)
			return
		}

		ready <- addr.String()
	}(ready)

	return p.Run(ctx)
}

type mysql struct {
	mysqlPath    string
	dir          string
	styledBranch string
	historyFile  string
	debug        bool
	printer      *printer.Printer
}

// Run runs the `mysql` client with the given arguments.
func (m *mysql) Run(ctx context.Context, args ...string) error {
	c := exec.CommandContext(ctx, m.mysqlPath, args...)
	if m.dir != "" {
		c.Dir = m.dir
	}

	c.Env = append(os.Environ(),
		fmt.Sprintf("MYSQL_HISTFILE=%s", m.historyFile),
	)

	c.Env = append(c.Env, fmt.Sprintf("MYSQL_PS1=%s", m.styledBranch))

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	return c.Run()
}

func formatMySQLBranch(database string, branch *ps.DatabaseBranch) string {
	branchStr := branch.Name

	if branch.Production {
		branchStr = fmt.Sprintf("|⚠ %s ⚠|", branch.Name)
	}

	return fmt.Sprintf("%s/%s> ", database, branchStr)
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
