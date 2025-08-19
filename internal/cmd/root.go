/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmd/dataimports"
	"github.com/planetscale/cli/internal/cmd/mcp"
	"github.com/planetscale/cli/internal/cmd/role"
	"github.com/planetscale/cli/internal/cmd/size"
	"github.com/planetscale/cli/internal/cmd/workflow"

	"github.com/fatih/color"
	"github.com/planetscale/cli/internal/cmd/api"
	"github.com/planetscale/cli/internal/cmd/auditlog"
	"github.com/planetscale/cli/internal/cmd/auth"
	"github.com/planetscale/cli/internal/cmd/backup"
	"github.com/planetscale/cli/internal/cmd/branch"
	"github.com/planetscale/cli/internal/cmd/connect"
	"github.com/planetscale/cli/internal/cmd/database"
	"github.com/planetscale/cli/internal/cmd/deployrequest"
	"github.com/planetscale/cli/internal/cmd/keyspace"
	"github.com/planetscale/cli/internal/cmd/org"
	"github.com/planetscale/cli/internal/cmd/password"
	"github.com/planetscale/cli/internal/cmd/ping"
	"github.com/planetscale/cli/internal/cmd/region"
	"github.com/planetscale/cli/internal/cmd/shell"
	"github.com/planetscale/cli/internal/cmd/signup"
	"github.com/planetscale/cli/internal/cmd/token"
	"github.com/planetscale/cli/internal/cmd/version"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/update"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	replacer = strings.NewReplacer("-", "_", ".", "_")
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:              "pscale",
	Short:            "A CLI for PlanetScale",
	Long:             `pscale is a CLI library for communicating with PlanetScale's API.`,
	TraverseChildren: true,
}

// Execute executes the command and returns the exit status of the finished
// command.
func Execute(ctx context.Context, sigc chan os.Signal, signals []os.Signal, ver, commit, buildDate string) int {
	var format printer.Format
	var debug bool

	if _, ok := os.LookupEnv("PSCALE_DISABLE_DEV_WARNING"); !ok {
		if commit == "" || ver == "" || buildDate == "" {
			fmt.Fprintf(os.Stderr, "!! WARNING: You are using a self-compiled binary which is not officially supported.\n!! To dismiss this warning, set PSCALE_DISABLE_DEV_WARNING=true\n\n")
		}
	}

	if update.Enabled() {
		updateCheckRes := make(chan *update.UpdateInfo, 1)
		updateCheckErr := make(chan error, 1)
		updateTimeout := time.After(500 * time.Millisecond)
		go func() {
			// note: don't close the chans to avoid
			// triggering the wrong select case
			ui, err := update.CheckVersion(ctx, ver)
			if err != nil {
				updateCheckErr <- err
			} else {
				updateCheckRes <- ui
			}
		}()
		defer func() {
			select {
			case ui := <-updateCheckRes:
				ui.PrintUpdateHint(ver)
			case err := <-updateCheckErr:
				fmt.Fprintf(os.Stderr, "Updater error: %v\n", err)
			case <-updateTimeout:
			}
		}()
	}

	err := runCmd(ctx, ver, commit, buildDate, &format, &debug, sigc, signals)
	if err == nil {
		return 0
	}

	// print any user specific messages first
	switch format {
	case printer.JSON:
		fmt.Fprintf(os.Stderr, `{"error": "%s"}`, err)
	default:
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}

	// check if a sub command wants to return a specific exit code
	var cmdErr *cmdutil.Error
	if errors.As(err, &cmdErr) {
		return cmdErr.ExitCode
	}

	return cmdutil.FatalErrExitCode
}

// runCmd adds all child commands to the root command, sets flags
// appropriately, and runs the root command.
func runCmd(ctx context.Context, ver, commit, buildDate string, format *printer.Format, debug *bool, sigc chan os.Signal, signals []os.Signal) error {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config",
		"", "Config file (default is $HOME/.config/planetscale/pscale.yml)")
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	v := version.Format(ver, commit, buildDate)
	rootCmd.SetVersionTemplate(v)
	rootCmd.Version = v
	rootCmd.Flags().Bool("version", false, "Show pscale version")

	cfg, err := config.New()
	if err != nil {
		return err
	}

	rootCmd.PersistentFlags().StringVar(&cfg.BaseURL,
		"api-url", ps.DefaultBaseURL, "The base URL for the PlanetScale API.")
	rootCmd.PersistentFlags().StringVar(&cfg.AccessToken,
		"api-token", cfg.AccessToken, "The API token to use for authenticating against the PlanetScale API.")

	rootCmd.PersistentFlags().VarP(printer.NewFormatValue(printer.Human, format), "format", "f",
		"Show output in a specific format. Possible values: [human, json, csv]")
	if err := viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format")); err != nil {
		return err
	}
	rootCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"human", "json", "csv"}, cobra.ShellCompDirectiveDefault
	})

	rootCmd.PersistentFlags().BoolVar(debug, "debug", false, "Enable debug mode")
	if err := viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug")); err != nil {
		return err
	}

	userAgent := "pscale-cli/" + ver
	headers := map[string]string{
		"pscale-cli-version": ver,
	}

	ch := &cmdutil.Helper{
		Printer:  printer.NewPrinter(format),
		Config:   cfg,
		ConfigFS: config.NewConfigFS(osFS{}),
		Client: func() (*ps.Client, error) {
			return cfg.NewClientFromConfig(ps.WithUserAgent(userAgent), ps.WithRequestHeaders(headers))
		},
	}
	ch.SetDebug(debug)

	// service token flags. they are hidden for now.
	rootCmd.PersistentFlags().StringVar(&cfg.ServiceTokenID,
		"service-token-name", "", "The Service Token name for authenticating.")
	rootCmd.PersistentFlags().StringVar(&cfg.ServiceTokenID, "service-token-id", "", "The Service Token ID for authenticating.")
	rootCmd.PersistentFlags().StringVar(&cfg.ServiceToken,
		"service-token", "", "Service Token for authenticating.")

	rootCmd.PersistentFlags().BoolVar(&color.NoColor, "no-color", false, "Disable color output")
	if err := viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color")); err != nil {
		return err
	}

	rootCmd.PersistentFlags().MarkDeprecated("service-token-name", "use --service-token-id instead")
	rootCmd.PersistentFlags().MarkHidden("service-token-name")

	// We don't want to show the default value
	rootCmd.PersistentFlags().Lookup("api-token").DefValue = ""

	// Add command groups for better organization
	rootCmd.AddGroup(&cobra.Group{ID: "database", Title: printer.Bold("General database commands:")})
	rootCmd.AddGroup(&cobra.Group{ID: "vitess", Title: printer.Bold("Vitess-specific commands:")})
	rootCmd.AddGroup(&cobra.Group{ID: "postgres", Title: printer.Bold("Postgres-specific commands:")})
	rootCmd.AddGroup(&cobra.Group{ID: "platform", Title: printer.Bold("Platform & account management:")})

	loginCmd := auth.LoginCmd(ch)
	loginCmd.Hidden = true
	logoutCmd := auth.LogoutCmd(ch)
	logoutCmd.Hidden = true

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)

	// Platform & Account Management commands
	apiCmd := api.ApiCmd(ch, userAgent, headers)
	apiCmd.GroupID = "platform"
	rootCmd.AddCommand(apiCmd)

	auditlogCmd := auditlog.AuditLogCmd(ch)
	auditlogCmd.GroupID = "platform"
	rootCmd.AddCommand(auditlogCmd)

	authCmd := auth.AuthCmd(ch)
	authCmd.GroupID = "platform"
	rootCmd.AddCommand(authCmd)

	completionCmd := CompletionCmd()
	completionCmd.GroupID = "platform"
	rootCmd.AddCommand(completionCmd)

	mcpCmd := mcp.McpCmd(ch)
	mcpCmd.GroupID = "database"
	rootCmd.AddCommand(mcpCmd)

	orgCmd := org.OrgCmd(ch)
	orgCmd.GroupID = "platform"
	rootCmd.AddCommand(orgCmd)

	pingCmd := ping.PingCmd(ch)
	pingCmd.GroupID = "platform"
	rootCmd.AddCommand(pingCmd)

	regionCmd := region.RegionCmd(ch)
	regionCmd.GroupID = "platform"
	rootCmd.AddCommand(regionCmd)

	signupCmd := signup.SignupCmd(ch)
	signupCmd.GroupID = "platform"
	rootCmd.AddCommand(signupCmd)

	sizeCmd := size.SizeCmd(ch)
	sizeCmd.GroupID = "platform"
	rootCmd.AddCommand(sizeCmd)

	tokenCmd := token.TokenCmd(ch)
	tokenCmd.GroupID = "platform"
	rootCmd.AddCommand(tokenCmd)

	versionCmd := version.VersionCmd(ch, ver, commit, buildDate)
	versionCmd.GroupID = "platform"
	rootCmd.AddCommand(versionCmd)

	// Database management commands (Both databases)
	backupCmd := backup.BackupCmd(ch)
	backupCmd.GroupID = "database"
	rootCmd.AddCommand(backupCmd)

	branchCmd := branch.BranchCmd(ch)
	branchCmd.GroupID = "database"
	rootCmd.AddCommand(branchCmd)

	databaseCmd := database.DatabaseCmd(ch)
	databaseCmd.GroupID = "database"
	rootCmd.AddCommand(databaseCmd)

	// Vitess-specific commands
	connectCmd := connect.ConnectCmd(ch)
	connectCmd.GroupID = "vitess"
	rootCmd.AddCommand(connectCmd)

	dataimportsCmd := dataimports.DataImportsCmd(ch)
	dataimportsCmd.GroupID = "vitess"
	rootCmd.AddCommand(dataimportsCmd)

	deployRequestCmd := deployrequest.DeployRequestCmd(ch)
	deployRequestCmd.GroupID = "vitess"
	rootCmd.AddCommand(deployRequestCmd)

	keyspaceCmd := keyspace.KeyspaceCmd(ch)
	keyspaceCmd.GroupID = "vitess"
	rootCmd.AddCommand(keyspaceCmd)

	passwordCmd := password.PasswordCmd(ch)
	passwordCmd.GroupID = "vitess"
	rootCmd.AddCommand(passwordCmd)

	shellCmd := shell.ShellCmd(ch, sigc, signals...)
	shellCmd.GroupID = "database"
	rootCmd.AddCommand(shellCmd)

	workflowCmd := workflow.WorkflowCmd(ch)
	workflowCmd.GroupID = "vitess"
	rootCmd.AddCommand(workflowCmd)

	// Postgres-specific commands
	roleCmd := role.RoleCmd(ch)
	roleCmd.GroupID = "postgres"
	rootCmd.AddCommand(roleCmd)

	return rootCmd.ExecuteContext(ctx)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		defaultConfigDir, err := config.ConfigDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(cmdutil.FatalErrExitCode)
		}

		// Order of preference for configuration files:
		// (1) $HOME/.config/planetscale
		viper.AddConfigPath(defaultConfigDir)
		viper.SetConfigName("pscale")
		viper.SetConfigType("yml")
	}

	viper.SetEnvPrefix("planetscale")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only handle errors when it's something unrelated to the config file not
			// existing.
			fmt.Println(err)
			os.Exit(cmdutil.FatalErrExitCode)
		}
	}

	// If no configFile is passed:
	// 1. Check local git repo for a config file
	// 2. If not in a git repo. Check working directory for a config file
	if cfgFile == "" {
		if rootDir, err := config.RootGitRepoDir(); err == nil {
			viper.AddConfigPath(rootDir)
			viper.SetConfigName(config.ProjectConfigFile())
			_ = viper.MergeInConfig()
		} else if localDir, err := config.LocalDir(); err == nil {
			viper.AddConfigPath(localDir)
			viper.SetConfigName(config.ProjectConfigFile())
			_ = viper.MergeInConfig()
		}
	}

	postInitCommands(rootCmd.Commands())
}

// Hacky fix for getting Cobra required flags and Viper playing well together.
// See: https://github.com/spf13/viper/issues/397
func postInitCommands(commands []*cobra.Command) {
	for _, cmd := range commands {
		presetRequiredFlags(cmd)
		if cmd.HasSubCommands() {
			postInitCommands(cmd.Commands())
		}
	}
}

func presetRequiredFlags(cmd *cobra.Command) {
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		log.Fatalf("error binding flags: %v", err)
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			err = cmd.Flags().Set(f.Name, viper.GetString(f.Name))
			if err != nil {
				log.Fatalf("error setting flag %s: %v", f.Name, err)
			}
		}
	})
}

// https://github.com/golang/go/issues/44286
type osFS struct{}

func (c osFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}
