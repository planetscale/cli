package dump

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/dumper"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/proxyutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/sql-proxy/proxy"
	"github.com/planetscale/sql-proxy/sigutil"

	"github.com/spf13/cobra"
)

type dumpFlags struct {
	localAddr string
	tables    string
	output    string
}

// DumpCmd encapsulates the commands for dumping a databse
func DumpCmd(ch *cmdutil.Helper) *cobra.Command {
	f := &dumpFlags{}
	cmd := &cobra.Command{
		Use:   "dump <database> <branch> [options]",
		Short: "Backup and dump your database",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE:  func(cmd *cobra.Command, args []string) error { return run(ch, cmd, f, args) },
	}

	cmd.PersistentFlags().StringVar(&f.localAddr, "local-addr",
		"", "Local address to bind and listen for connections. By default the proxy binds to 127.0.0.1 with a random port.")
	cmd.PersistentFlags().StringVar(&f.tables, "tables", "", "Comma separated string of tables to dump. By default all tables are dumped.")
	cmd.PersistentFlags().StringVar(&f.output, "output", "", "Output director of the dump. By default the dump is stored in the current director.")

	return cmd
}

func run(ch *cmdutil.Helper, cmd *cobra.Command, flags *dumpFlags, args []string) error {
	ctx := context.Background()
	database := args[0]
	branch := args[1]

	client, err := ch.Config.NewClientFromConfig()
	if err != nil {
		return err
	}

	const localProxyAddr = "127.0.0.1"
	localAddr := localProxyAddr + ":0"
	if flags.localAddr != "" {
		localAddr = flags.localAddr
	}

	proxyOpts := proxy.Options{
		CertSource: proxyutil.NewRemoteCertSource(client),
		LocalAddr:  localAddr,
		Instance:   fmt.Sprintf("%s/%s/%s", ch.Config.Organization, database, branch),
		Logger:     cmdutil.NewZapLogger(ch.Debug),
	}

	p, err := proxy.NewClient(proxyOpts)
	if err != nil {
		return fmt.Errorf("couldn't create proxy client: %s", err)
	}

	// TODO(fatih): replace with signal.NotifyContext once Go 1.16 is released
	// https://go-review.googlesource.com/c/go/+/219640
	ctx = sigutil.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err := p.Run(ctx)
		if err != nil {
			ch.Printer.Println("proxy error: ", err)
		}
	}()

	status, err := client.DatabaseBranches.GetStatus(ctx, &ps.GetDatabaseBranchStatusRequest{
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

	if status.Credentials.User == "" {
		return errors.New("database branch is not ready yet, please try again in a few minutes.")
	}

	addr, err := p.LocalAddr()
	if err != nil {
		return err
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if flags.output != "" {
		dir = flags.output
	}

	dumperCfg := dumper.NewDefaultConfig()
	dumperCfg.User = status.Credentials.User
	dumperCfg.Password = status.Credentials.Password
	dumperCfg.Address = addr.String()
	dumperCfg.Database = database
	if flags.tables != "" {
		dumperCfg.Table = flags.tables
	}
	dumperCfg.Database = database
	dumperCfg.Outdir = dir

	d, err := dumper.NewDumper(dumperCfg)
	if err != nil {
		return err
	}

	if flags.tables == "" {
		ch.Printer.Printf("Starting to dump all tables from database %s to dir %q",
			printer.BoldBlue(database), printer.Bold(dir))
	} else {
		ch.Printer.Printf("Starting to dump tables %q from database %s to dir %q",
			printer.BoldRed(flags.tables), printer.BoldBlue(database), printer.BoldBlue(dir))
	}

	err = d.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to dump database: %s", err)
	}

	return nil
}

// [mysql]
// # The host to connect to
// host = 127.0.0.1
// # TCP/IP port to conect to
// port = 3306
// # Username with privileges to run the dump
// user = root
// # User password
// password = [REDACTED]
// # Database to dump
// database = planetscale
// # Table(s) to dump ;  comment out to dump all tables in database
// ;table = corder,product
// # Directory to dump files to
// outdir = ./dumper-sql
// # Split tables into chunks of this output file size. This value is in MB
// chunksize = 128
// # Session variables, split by ;
// # vars= "xx=xx;xx=xx;"
// # The workload variable here is required for Vitess to use streaming SELECTs
// #   if we don't use streaming selects, we'll run into row limits.
// vars=set workload=olap;
// # Format to dump:
// #  mysql - MySQL inserts (default)
// #  tsv   - TSV format
// #  csv   - CSV format
// format = mysql
// # Use this to use regexp to control what databases to export. These are optional
// [database]
// # regexp = ^(mysql|sys|information_schema|performance_schema)$
// # As the used regexp lib does not allow for lookarounds, you may use this to invert the whole regexp
// # This option should be refactored as soon as a GPLv3 compliant go-pcre lib is found
// # invert_regexp = on
// # Use this to restrict exported data. These are optional
// [where]
// # sample_table1 = created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
// # sample_table2 = created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
// # Use this to override value returned from tables. These are optional
// [select]
// # customer.first_name = CONCAT('Bohu', id)
// # customer.last_name = 'Last'
// # Use this to ignore the column to dump.
// [filter]
// # table1.column1 = ignore
