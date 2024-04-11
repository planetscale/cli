package shell

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/mitchellh/go-homedir"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
	exec "golang.org/x/sys/execabs"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/passwordutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/promptutil"
	"github.com/planetscale/cli/internal/proxyutil"
)

func ShellCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
		role       string
		replica    bool
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

			replica := flags.replica

			role := cmdutil.AdministratorRole
			if flags.role != "" {
				role, err = cmdutil.RoleFromString(flags.role)
				if err != nil {
					return err
				}
			} else if replica {
				role = cmdutil.ReaderRole
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
			pw, err := passwordutil.New(ctx, client, passwordutil.Options{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Role:         role,
				Name:         passwordutil.GenerateName("pscale-cli-shell"),
				TTL:          5 * time.Minute,
				Replica:      replica,
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

			proxyAddr := l.Addr().String()
			host, port, err := net.SplitHostPort(proxyAddr)
			if err != nil {
				return cmdutil.HandleError(err)
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
			if replica {
				mysqlArgs = append([]string{"--no-defaults"}, mysqlArgs...)
			} else {
				mysqlArgs = append(mysqlArgs, "-D", "@primary")
			}

			historyFile := historyFilePath(ch.Config.Organization, database, branch)
			styledBranch := formatMySQLBranch(database, dbBranch)

			m := &mysql{
				mysqlPath:    mysqlPath,
				historyFile:  historyFile,
				styledBranch: styledBranch,
				debug:        ch.Debug(),
				printer:      ch.Printer,
			}

			errCh := make(chan error, 1)
			go func() {
				errCh <- proxy.Serve(l)
			}()

			go func() {
				errCh <- m.Run(ctx, mysqlArgs...)
			}()

			go func() {
				errCh <- pw.Renew(ctx)
			}()

			select {
			case <-ctx.Done():
				return nil
			case err := <-errCh:
				if err == nil {
					return nil
				}
				return cmdutil.HandleError(err)
			}
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&flags.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API. (format: `hostname:port`)")
	cmd.PersistentFlags().StringVar(&flags.role, "role",
		"", "Role defines the access level, allowed values are: reader, writer, readwriter, admin. Defaults to 'reader' for replica passwords, otherwise defaults to 'admin'.")
	cmd.Flags().BoolVar(&flags.replica, "replica", false, "When enabled, the password will route all reads to the branch's primary replicas and all read-only regions.")

	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
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

// Originally we wrote history to the home directory, if present, keep using it
func legacyHistoryFilePath(org, db, branch string) string {
	dir, err := homedir.Dir()
	if err != nil {
		return ""
	}

	historyDir := filepath.Join(dir, ".pscale", "history")

	_, err = os.Stat(historyDir)
	if os.IsNotExist(err) {
		return ""
	}

	historyFilename := fmt.Sprintf("%s.%s.%s", org, db, branch)
	historyFile := filepath.Join(historyDir, historyFilename)

	return historyFile
}

func historyFilePath(org, db, branch string) string {
	legacyHistoryFile := legacyHistoryFilePath(org, db, branch)
	if legacyHistoryFile != "" {
		return legacyHistoryFile
	}

	historyFilePath := fmt.Sprintf(".pscale/history/%s.%s.%s", org, db, branch)

	historyFile, err := xdg.DataFile(historyFilePath)
	if err != nil {
		return ""
	}

	return historyFile
}
