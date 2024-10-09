package connect

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/passwordutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/promptutil"
	"github.com/planetscale/cli/internal/proxyutil"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"

	"vitess.io/vitess/go/mysql"
)

func ConnectCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		port                string
		host                string
		remoteAddr          string
		execCommand         string
		execCommandProtocol string
		execCommandEnvURL   string
		role                string
		noRandom            bool
		replica             bool
		authMethod          string
	}

	cmd := &cobra.Command{
		Use: "connect [database] [branch]",
		// we only require database, because we deduct branch automatically
		Args:  cmdutil.RequiredArgs("database"),
		Short: "Create a secure connection to a database and branch for a local client",
		Example: `The connect subcommand establishes a secure connection between your host and PlanetScale.

By default, if no branch names are given and there is only one branch, it
automatically connects to that branch:

  pscale connect mydatabase

If there are multiple branches for the given database, you'll be prompted to
choose one. To connect to a specific branch, pass the branch as a second
argument:

  pscale connect mydatabase mybranch`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			database := args[0]

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
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrNotFound:
						return fmt.Errorf("database %s does not exist in organization %s",
							printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
					default:
						return cmdutil.HandleError(err)
					}
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

			authMethod := mysql.CachingSha2Password
			if flags.authMethod != "" {
				switch flags.authMethod {
				case "caching_sha2_password":
					authMethod = mysql.CachingSha2Password
				case "mysql_native_password":
					authMethod = mysql.MysqlNativePassword
				default:
					return fmt.Errorf("unsupported auth method: %s", flags.authMethod)
				}
			}

			// check whether database and branch exist
			dbBranch, err := client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s and branch %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
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
				Name:         passwordutil.GenerateName("pscale-cli-connect"),
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

			l, err := listenProxy(ch, flags.host, flags.port, !flags.noRandom)
			if err != nil {
				return cmdutil.HandleError(err)
			}
			defer l.Close()

			errCh := make(chan error, 1)
			go func() {
				errCh <- proxy.Serve(l, authMethod)
			}()

			go func() {
				errCh <- pw.Renew(ctx)
			}()

			localAddr := l.Addr().String()

			ch.Printer.Printf("Secure connection to database %s and branch %s is established!.\n\nLocal address to connect your application: %s (press ctrl-c to quit)\n",
				printer.BoldBlue(database),
				printer.BoldBlue(branch),
				printer.BoldBlue(localAddr),
			)

			if flags.execCommand != "" {
				go func() {
					errCh <- runCommand(
						ctx,
						localAddr,
						flags.execCommand,
						flags.execCommandProtocol,
						flags.execCommandEnvURL,
						database,
						branch,
					)

					// TODO(fatih): is it worth to making cancellation configurable?
					cancel() // stop the proxy by cancelling all other child contexts
				}()
			}

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
	cmd.PersistentFlags().StringVar(&flags.host, "host", "127.0.0.1", "Local host to bind and listen for connections")
	cmd.PersistentFlags().StringVar(&flags.port, "port", "3306", "Local port to bind and listen for connections")
	cmd.PersistentFlags().BoolVar(&flags.noRandom, "no-random", false, "Do not pick a random port if the default port is in use")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API.")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck
	cmd.PersistentFlags().StringVar(&flags.execCommand, "execute", "", "Run this command after successfully connecting to the database.")
	cmd.PersistentFlags().StringVar(&flags.execCommandProtocol, "execute-protocol",
		"mysql2", "Protocol for the exposed URL (by default DATABASE_URL) value in execute")
	cmd.PersistentFlags().StringVar(&flags.execCommandEnvURL, "execute-env-url", "DATABASE_URL",
		"Environment variable name that contains the exposed Database URL.")
	cmd.PersistentFlags().StringVar(&flags.role, "role",
		"", "Role defines the access level, allowed values are: reader, writer, readwriter, admin. Defaults to 'reader' for replica passwords, otherwise defaults to 'admin'.")
	cmd.Flags().BoolVar(&flags.replica, "replica", false, "When enabled, the password will route all reads to the branch's primary replicas and all read-only regions.")
	cmd.PersistentFlags().StringVar(&flags.authMethod, "mysql-auth-method",
		"", "MySQL auth method defines the authentication method returned for the MySQL protocol. Allowed values are: caching_sha2_password, mysql_native_password. Defaults to 'caching_sha2_password'.")

	return cmd
}

func listenProxy(ch *cmdutil.Helper, host, port string, random bool) (net.Listener, error) {
	addr := net.JoinHostPort(host, port)
	l, err := net.Listen("tcp", addr)
	if err == nil {
		return l, nil
	}
	if random && isAddrInUse(err) {
		ch.Printer.Printf("Tried address %s, but it's already in use. Picking up a random port ...\n", addr)
		return listenProxy(ch, host, "0", false)
	}
	return nil, err
}

// runCommand runs the given command with several environment variables exposed
// to the command.
func runCommand(ctx context.Context, addr, command, protocol, databaseEnvURL, database, branch string) error {
	args, err := shellwords.Parse(command)
	if err != nil {
		return fmt.Errorf("failed to parse command, not running: %s", err)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	connStr := fmt.Sprintf("%s=%s://root@%s/%s", databaseEnvURL, protocol, addr, database)
	cmd.Env = append(cmd.Env, connStr)

	hostEnv := fmt.Sprintf("PLANETSCALE_DATABASE_HOST=%s", addr)
	cmd.Env = append(cmd.Env, hostEnv)

	dbName := fmt.Sprintf("PLANETSCALE_DATABASE_NAME=%s", database)
	cmd.Env = append(cmd.Env, dbName)

	branchName := fmt.Sprintf("PLANETSCALE_BRANCH_NAME=%s", branch)
	cmd.Env = append(cmd.Env, branchName)

	err = cmd.Run()
	if err == nil {
		return nil
	}

	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return &cmdutil.Error{
			Msg:      fmt.Sprintf("running command with --execute has failed: %s\n", err),
			ExitCode: ee.ProcessState.ExitCode(),
		}
	}

	return err
}

// isAddrInUse returns an error if the error indicates that the given address
// is already in use. Becaue different OS return different error messages, we
// try to get the underlying error.
// see: https://stackoverflow.com/a/65865898
func isAddrInUse(err error) bool {
	var syserr *os.SyscallError
	if !errors.As(err, &syserr) {
		return false
	}
	var errErrno syscall.Errno // doesn't need a "*" (ptr) because it's already a ptr (uintptr)
	if !errors.As(syserr, &errErrno) {
		return false
	}
	if errErrno == syscall.EADDRINUSE {
		return true
	}

	if runtime.GOOS == "windows" {
		// Looks like on Windows we might see multiple different errors. Here
		// are some information on how to track them down.
		// There is a list of human readable Windows errors here:  https://github.com/benmoss/monitor/blob/8bcb512752ea0d322e0498309ab2cc1821090f01/errno/msg.go
		// The errors constants map to the syscall codes defined in: https://pkg.go.dev/golang.org/x/sys/windows
		// The official docs for some windows socket errors are here: // https://docs.microsoft.com/en-us/windows/win32/winsock/windows-sockets-error-codes-2
		const (
			WSAEACCES     = 10013
			WSAEADDRINUSE = 10048
		)

		switch errErrno {
		case WSAEACCES, WSAEADDRINUSE:
			return true
		}
	}

	return false
}
