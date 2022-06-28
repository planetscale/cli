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

	"github.com/planetscale/cli/internal/cmd/dataimports"

	"github.com/fatih/color"
	"github.com/planetscale/cli/internal/cmd/auditlog"
	"github.com/planetscale/cli/internal/cmd/auth"
	"github.com/planetscale/cli/internal/cmd/backup"
	"github.com/planetscale/cli/internal/cmd/branch"
	"github.com/planetscale/cli/internal/cmd/connect"
	"github.com/planetscale/cli/internal/cmd/database"
	"github.com/planetscale/cli/internal/cmd/deployrequest"
	"github.com/planetscale/cli/internal/cmd/org"
	"github.com/planetscale/cli/internal/cmd/password"
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
func Execute(ctx context.Context, ver, commit, buildDate string) int {
	var format printer.Format
	var debug bool

	if _, ok := os.LookupEnv("PSCALE_DISABLE_DEV_WARNING"); !ok {
		if commit == "" || ver == "" || buildDate == "" {
			fmt.Fprintf(os.Stderr, "!! WARNING: You are using a self-compiled binary which is not officially supported.\n!! To dismiss this warning, set PSCALE_DISABLE_DEV_WARNING=true\n\n")
		}
	}

	err := runCmd(ctx, ver, commit, buildDate, &format, &debug)
	if err == nil {
		return 0
	}

	// print any user specific messages first
	switch format {
	case printer.JSON:
		fmt.Fprintf(os.Stderr, `{"error": "%s"}`, err)
	default:
		if err := update.CheckVersion(ctx, ver); err != nil && debug {
			fmt.Fprintf(os.Stderr, "Updater error: %s\n", err)
		}

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
func runCmd(ctx context.Context, ver, commit, buildDate string, format *printer.Format, debug *bool) error {
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

	ch := &cmdutil.Helper{
		Printer:  printer.NewPrinter(format),
		Config:   cfg,
		ConfigFS: config.NewConfigFS(osFS{}),
		Client: func() (*ps.Client, error) {
			return cfg.NewClientFromConfig()
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

	loginCmd := auth.LoginCmd(ch)
	loginCmd.Hidden = true
	logoutCmd := auth.LogoutCmd(ch)
	logoutCmd.Hidden = true

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(auditlog.AuditLogCmd(ch))
	rootCmd.AddCommand(auth.AuthCmd(ch))
	rootCmd.AddCommand(backup.BackupCmd(ch))
	rootCmd.AddCommand(branch.BranchCmd(ch))
	rootCmd.AddCommand(connect.ConnectCmd(ch))
	rootCmd.AddCommand(database.DatabaseCmd(ch))
	rootCmd.AddCommand(dataimports.DataImportsCmd(ch))
	rootCmd.AddCommand(deployrequest.DeployRequestCmd(ch))
	rootCmd.AddCommand(org.OrgCmd(ch))
	rootCmd.AddCommand(password.PasswordCmd(ch))
	rootCmd.AddCommand(region.RegionCmd(ch))
	rootCmd.AddCommand(shell.ShellCmd(ch))
	rootCmd.AddCommand(signup.SignupCmd(ch))
	rootCmd.AddCommand(token.TokenCmd(ch))
	rootCmd.AddCommand(version.VersionCmd(ch, ver, commit, buildDate))

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

	// Check for a project-local configuration file to merge in if the user
	// has not specified a config file
	if rootDir, err := config.RootGitRepoDir(); err == nil && cfgFile == "" {
		viper.AddConfigPath(rootDir)
		viper.SetConfigName(config.ProjectConfigFile())
		viper.MergeInConfig() // nolint:errcheck
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
