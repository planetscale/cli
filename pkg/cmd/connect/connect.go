package connect

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func ConnectCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
		verbose    bool
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
				branch, err = fetchBranch(ctx, client, cfg.Organization, database)
				if err != nil {
					return err
				}
			}

			instance := fmt.Sprintf("%s/%s/%s", cfg.Organization, database, branch)

			certSource := &remoteCertSource{client: client}

			proxyOpts := proxy.Options{
				CertSource: certSource,
				LocalAddr:  flags.localAddr,
				RemoteAddr: flags.remoteAddr,
				Instance:   instance,
			}

			if !flags.verbose {
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
	cmd.PersistentFlags().BoolVar(&flags.verbose, "v", false, "enable verbose mode")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}

type remoteCertSource struct {
	client *ps.Client
}

func (r *remoteCertSource) Cert(ctx context.Context, org, db, branch string) (*proxy.Cert, error) {
	pkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate private key: %s", err)
	}

	cert, err := r.client.Certificates.Create(ctx, &ps.CreateCertificateRequest{
		Organization: org,
		DatabaseName: db,
		Branch:       branch,
		PrivateKey:   pkey,
	})
	if err != nil {
		return nil, err
	}

	return &proxy.Cert{
		ClientCert: cert.ClientCert,
		CACert:     cert.CACert,
		RemoteAddr: cert.RemoteAddr,
	}, nil
}

func fetchBranch(ctx context.Context, client *planetscale.Client, org, db string) (string, error) {
	branches, err := client.DatabaseBranches.List(ctx, &planetscale.ListDatabaseBranchesRequest{
		Organization: org,
		Database:     db,
	})
	if err != nil {
		return "", err
	}

	if len(branches) == 0 {
		return "", fmt.Errorf("no branch exist for database: %q", db)
	}

	// if there is only one branch, just return it
	if len(branches) == 1 {
		return branches[0].Name, nil
	}

	branchNames := make([]string, 0, len(branches)-1)
	for _, b := range branches {
		branchNames = append(branchNames, b.Name)
	}

	prompt := &survey.Select{
		Message: "Select a branch to connect to:",
		Options: branchNames,
		VimMode: true,
	}

	type result struct {
		branch string
		err    error
	}

	resp := make(chan result)

	go func() {
		var branch string
		err := survey.AskOne(prompt, &branch)
		resp <- result{
			branch: branch,
			err:    err,
		}
	}()

	// timeout so CLI is not blocked forever if the user accidently called it
	select {
	case <-time.After(time.Second * 20):
		// TODO(fatih): this is buggy. Because there is no proper cancellation
		// in the survey.AskOne() function, it holds to stdin, which causes the
		// terminal to malfunction. But the timeout is not intended for regular
		// users, it's meant to catch script invocations, so let's still use it
		return "", errors.New("pscale connect timeout: no branch is selected")
	case r := <-resp:
		return r.branch, r.err
	}
}
