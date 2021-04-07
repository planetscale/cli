package connect

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/promptutil"
	"github.com/planetscale/cli/internal/proxyutil"

	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func ConnectCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
		debug      bool
	}

	cmd := &cobra.Command{
		Use: "connect [database] [branch]",
		// we only require database, because we deduct branch automatically
		Args:  cmdutil.RequiredArgs("database"),
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
			database := args[0]

			if !cmdutil.IsTTY || ch.Printer.Format() != printer.Human {
				return errors.New("pscale connect only works in interactive mode")
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

			localAddr := "127.0.0.1:3306"
			if flags.localAddr != "" {
				localAddr = flags.localAddr
			}

			proxyOpts := proxy.Options{
				CertSource: proxyutil.NewRemoteCertSource(client),
				LocalAddr:  localAddr,
				RemoteAddr: flags.remoteAddr,
				Instance:   fmt.Sprintf("%s/%s/%s", ch.Config.Organization, database, branch),
			}

			if !flags.debug {
				proxyOpts.Logger = zap.NewNop()
			}

			err = runProxy(proxyOpts, database, branch)
			if err != nil {
				if isAddrInUse(err) {
					ch.Printer.Println("Tried address 127.0.0.1:3306, but it's already in use. Picking up a random port ...")
					proxyOpts.LocalAddr = "127.0.0.1:0"
					return runProxy(proxyOpts, database, branch)
				}
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&flags.localAddr, "local-addr", "",
		"Local address to bind and listen for connections")
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "",
		"PlanetScale Database remote network address. By default the remote address is populated automatically from the PlanetScale API.")
	cmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "enable debug mode")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}

func runProxy(proxyOpts proxy.Options, database, branch string) error {
	ctx := context.Background()
	p, err := proxy.NewClient(proxyOpts)
	if err != nil {
		return fmt.Errorf("couldn't create proxy client: %s", err)
	}

	go func() {
		// this is blocking and will only return once p.Run() below is
		// invoked
		addr, err := p.LocalAddr()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed getting local addr: %s\n", err)
			return
		}

		fmt.Printf("Secure connection to databases %s and branch %s is established!.\n\nLocal address to connect your application: %s (press ctrl-c to quit)",
			cmdutil.BoldBlue(database),
			cmdutil.BoldBlue(branch),
			cmdutil.BoldBlue(addr.String()),
		)
	}()

	// TODO(fatih): replace with signal.NotifyContext once Go 1.16 is released
	// https://go-review.googlesource.com/c/go/+/219640
	ctx = sigutil.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
	return p.Run(ctx)
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

	const WSAEADDRINUSE = 10048
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}
	return false
}
