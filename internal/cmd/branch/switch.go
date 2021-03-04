package branch

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var tl = []string{"rev-parse", "--show-toplevel"}

const psdbFile = ".psdb"

func SwitchCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <database> <branch>",
		Short: "Switches the current project to use the specified branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) != 2 {
				return cmd.Usage()
			}

			database, branch := args[0], args[1]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			b, err := client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
				Organization: cfg.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil && !errorIsNotFound(err) {
				return err
			}

			if errorIsNotFound(err) {
				createReq := &ps.CreateDatabaseBranchRequest{
					Organization: cfg.Organization,
					Database:     database,
					Branch: &ps.DatabaseBranch{
						Name:         branch,
						ParentBranch: "main", // todo(nickvanw): can we discern this?
					},
				}

				b, err = client.DatabaseBranches.Create(ctx, createReq)
				if err != nil {
					return err
				}
			}

			rootPath, err := getRootDir()
			if err != nil {
				return err
			}

			cfgFile := filepath.Join(rootPath, psdbFile)

			f, err := os.OpenFile(cfgFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			cfg := projectConfig{
				Database: database,
				Branch:   b.Name,
			}

			return yaml.NewEncoder(f).Encode(cfg)
		},
	}

	return cmd
}

type projectConfig struct {
	Database string `yaml:"database"`
	Branch   string `yaml:"branch"`
}

func errorIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == http.StatusText(http.StatusNotFound)
}

func getRootDir() (string, error) {
	out, err := exec.Command("git", tl...).CombinedOutput()
	if err != nil {
		return os.Getwd()
	}

	return string(strings.TrimSuffix(string(out), "\n")), nil
}
