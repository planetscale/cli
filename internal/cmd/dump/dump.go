package dump

import (
	"errors"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/dumper"

	"github.com/spf13/cobra"
)

// DumpCmd encapsulates the commands for dumping a databse
func DumpCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump <database> <branch> [options]",
		Short: "Backup and dump your database",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE:  func(cmd *cobra.Command, args []string) error { return run(ch, cmd, args) },
	}
	return cmd
}

func run(ch *cmdutil.Helper, cmd *cobra.Command, args []string) error {
	db := args[0]
	branch := args[1]

	fmt.Printf("db = %+v\n", db)
	fmt.Printf("branch = %+v\n", branch)

	cfg := &dumper.Config{}
	d, err := dumper.NewDumper(cfg)
	if err != nil {
		return err
	}

	err = d.Run()
	if err != nil {
		return err
	}

	// pscale dump planetscale main --tables users
	return errors.New("not implemented yet")
}
