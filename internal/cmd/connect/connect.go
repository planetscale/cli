package connect

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/promptutil"
	"github.com/planetscale/cli/internal/proxyutil"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/mattn/go-shellwords"
	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"
	"github.com/spf13/cobra"
)

func ConnectCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		port                string
		host                string
		remoteAddr          string
		execCommand         string
		execCommandProtocol string
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
			ctx := context.Background()
			database := args[0]

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
			_, err = client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
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

			localAddr := net.JoinHostPort(flags.host, flags.port)

			proxyOpts := proxy.Options{
				CertSource: proxyutil.NewRemoteCertSource(client),
				LocalAddr:  localAddr,
				RemoteAddr: flags.remoteAddr,
				Instance:   fmt.Sprintf("%s/%s/%s", ch.Config.Organization, database, branch),
				Logger:     cmdutil.NewZapLogger(ch.Debug()),
			}

			proxyReady := make(chan string, 1)

			if flags.execCommand != "" {
				go runCommand(ctx, flags.execCommand, flags.execCommandProtocol, database, branch, proxyReady)
			}

			err = runProxy(proxyOpts, database, branch, proxyReady)
			if err != nil {
				if isAddrInUse(err) {
					ch.Printer.Printf("Tried address %s, but it's already in use. Picking up a random port ...", localAddr)
					proxyOpts.LocalAddr = net.JoinHostPort(flags.host, "0")
					return runProxy(proxyOpts, database, branch, proxyReady)
				}
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&flags.host, "host", "127.0.0.1", "Local host to bind and listen for connections")
	cmd.PersistentFlags().StringVar(&flags.port, "port", "3306", "Local port to bind and listen for connections")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API.")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck
	cmd.PersistentFlags().StringVar(&flags.execCommand, "execute", "", "Run this command after successfully connecting to the database.")
	cmd.PersistentFlags().StringVar(&flags.execCommandProtocol, "execute-protocol", "mysql2", "Protocol for the DATABASE_URL value in execute")

	return cmd
}

func runProxy(proxyOpts proxy.Options, database, branch string, ready chan string) error {
	ctx := context.Background()
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

		fmt.Printf("Secure connection to database %s and branch %s is established!.\n\nLocal address to connect your application: %s (press ctrl-c to quit)",
			printer.BoldBlue(database),
			printer.BoldBlue(branch),
			printer.BoldBlue(addr.String()),
		)
		ready <- addr.String()
	}(ready)

	// TODO(fatih): replace with signal.NotifyContext once Go 1.16 is released
	// https://go-review.googlesource.com/c/go/+/219640
	ctx = sigutil.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
	return p.Run(ctx)
}

func runCommand(ctx context.Context, command, protocol, database, branch string, ready chan string) {
	args, err := shellwords.Parse(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nfailed to parse command, not running: %s", err)
		return
	}
	addr := <-ready

	ctx = sigutil.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
	child := exec.CommandContext(ctx, args[0], args[1:]...)
	child.Env = os.Environ()
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	connStr := fmt.Sprintf("DATABASE_URL=%s://root@%s/%s", protocol, addr, database)
	child.Env = append(child.Env, connStr)

	hostEnv := fmt.Sprintf("PLANETSCALE_DATABASE_HOST=%s", addr)
	child.Env = append(child.Env, hostEnv)

	dbName := fmt.Sprintf("PLANETSCALE_DATABASE_NAME=%s", database)
	child.Env = append(child.Env, dbName)

	branchName := fmt.Sprintf("PLANETSCALE_BRANCH_NAME=%s", branch)
	child.Env = append(child.Env, branchName)

	// todo(nickvanw): right now, this starts the process and then doesn't track what happens next
	// if the process exits (non-zero or otherwise) the proxy will remain running
	//
	// the right behavior is probably to at least allow the user to configure that the proxy should
	// exit when this process exits, and pass through the exit code
	if err := child.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "\nfailed to execute command: %s", err)
	}
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
