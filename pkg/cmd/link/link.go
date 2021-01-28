package link

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"syscall"

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

func LinkCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		localAddr  string
		remoteAddr string
		verbose    bool
	}

	cmd := &cobra.Command{
		Use:   "link [database] [branch]",
		Short: "Create a secure connection to the given database and branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) != 1 {
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
				end := cmdutil.PrintLinkProgress(
					fmt.Sprintf("🔐 Secure link to branch %s is established!. Connect to %s ... (press ctrl-c to quit)",
						cmdutil.BoldBlue(branch),
						cmdutil.BoldBlue(flags.localAddr),
					),
				)
				defer end()
				proxyOpts.Logger = zap.NewNop()
			}

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
	cmd.PersistentFlags().StringVar(&flags.remoteAddr, "remote-addr", "ac001fde9cdb746988cf56648d20f3d0-cc3d7d661b5e5955.elb.us-east-1.amazonaws.com:3306",
		"PlanetScale Database remote network address")
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

	// if there is only one branch, just return it
	if len(branches) == 1 {
		return branches[0].Name, nil
	}

	branchNames := make([]string, 0, len(branches)-1)
	for _, b := range branches {
		branchNames = append(branchNames, b.Name)
	}

	prompt := &survey.Select{
		Message: "Select a branch to connect",
		Options: branchNames,
	}

	var branch string
	err = survey.AskOne(prompt, &branch)
	if err != nil {
		return "", err
	}

	return branch, nil
}
