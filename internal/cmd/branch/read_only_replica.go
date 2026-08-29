package branch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ReadOnlyReplicaCmd manages read-only replicas of a branch.
func ReadOnlyReplicaCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-only-replica <command>",
		Short: "Create, list, and manage read-only replicas of a branch",
	}

	cmd.AddCommand(
		ReadOnlyReplicaListCmd(ch),
		ReadOnlyReplicaCreateCmd(ch),
		ReadOnlyReplicaShowCmd(ch),
		ReadOnlyReplicaDeleteCmd(ch),
		ReadOnlyReplicaChangesCmd(ch),
	)
	return cmd
}

func ReadOnlyReplicaListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <database> <branch>",
		Short: "List read-only replicas for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching read-only replicas for %s branch in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()
			replicas, err := client.ReadOnlyReplicas.List(ctx, &ps.ListReadOnlyReplicasRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, "read_branch",
						"branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(replicas) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No read-only replicas exist in branch %s of database %s.\n", printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}
			return ch.Printer.PrintResource(toReadOnlyReplicas(replicas))
		},
	}
	return cmd
}

func ReadOnlyReplicaCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name        string
		region      string
		replicas    int
		clusterSize string
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a read-only replica for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			req := &ps.CreateReadOnlyReplicaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Name:         flags.name,
				Region:       flags.region,
				ClusterSize:  flags.clusterSize,
			}
			if cmd.Flags().Changed("replicas") {
				req.Replicas = &flags.replicas
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating read-only replica %s for %s branch in %s...", printer.BoldBlue(flags.name), printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()
			replica, err := client.ReadOnlyReplicas.Create(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, "write_database",
						"branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()
			return ch.Printer.PrintResource(toReadOnlyReplica(replica))
		},
	}
	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the read-only replica")
	cmd.Flags().StringVar(&flags.region, "region", "", "Region in which to create the read-only replica")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Number of replicas (defaults to 1)")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "Cluster size (defaults to the primary's cluster size)")
	cmd.MarkFlagRequired("name")   // nolint:errcheck
	cmd.MarkFlagRequired("region") // nolint:errcheck
	return cmd
}

func ReadOnlyReplicaShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <database> <branch> <replica>",
		Short: "Show a read-only replica of a branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "replica"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching read-only replica %s for %s branch in %s...", printer.BoldBlue(name), printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()
			replica, err := client.ReadOnlyReplicas.Get(ctx, &ps.GetReadOnlyReplicaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Name:         name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, "read_branch",
						"read-only replica %s does not exist in branch %s of database %s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()
			return ch.Printer.PrintResource(toReadOnlyReplica(replica))
		},
	}
}

func ReadOnlyReplicaDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <database> <branch> <replica>",
		Short: "Delete a read-only replica of a branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "replica"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf(`cannot delete read-only replica with the output format "%s" (run with --force to override)`, ch.Printer.Format())
				}
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of read-only replica %q (run with --force to override)", name)
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", printer.Bold("Please type"), printer.BoldBlue(name), printer.Bold("to confirm:"))
				prompt := &survey.Input{Message: confirmationMessage}
				var userInput string
				err = survey.AskOne(prompt, &userInput)
				if err != nil {
					if err == terminal.InterruptErr {
						os.Exit(0)
					}
					return err
				}
				if userInput != name {
					return errors.New("incorrect read-only replica name entered, skipping read-only replica deletion")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting read-only replica %s from %s branch in %s...", printer.BoldBlue(name), printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()
			err = client.ReadOnlyReplicas.Delete(ctx, &ps.DeleteReadOnlyReplicaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Name:         name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, "write_database",
						"read-only replica %s does not exist in branch %s of database %s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Read-only replica %s was successfully deleted from %s branch in %s.\n", printer.BoldBlue(name), printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}
			return ch.Printer.PrintResource(map[string]string{"result": "read-only replica deleted"})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Delete a read-only replica without confirmation")
	return cmd
}

func ReadOnlyReplicaChangesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
		period  string
	}

	cmd := &cobra.Command{
		Use:   "changes <database> <branch>",
		Short: "List read-only replica change requests for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			opts := make([]ps.ListOption, 0, 2)
			if cmd.Flags().Changed("page") {
				opts = append(opts, ps.WithPage(flags.page))
			}
			if cmd.Flags().Changed("per-page") {
				opts = append(opts, ps.WithPerPage(flags.perPage))
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching read-only replica changes for %s branch in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()
			changes, err := client.ReadOnlyReplicas.ListChanges(ctx, &ps.ListReadOnlyReplicaChangesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Period:       flags.period,
			}, opts...)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, "read_branch",
						"branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(changes) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No read-only replica changes exist in branch %s of database %s.\n", printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}
			return ch.Printer.PrintResource(toReadOnlyReplicaChanges(changes))
		},
	}
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	cmd.Flags().StringVar(&flags.period, "period", "", "Time period to look back: 15m, 1h, 3h, 6h, 12h, 1d, 2d, 7d, or 8d")
	return cmd
}

type ReadOnlyReplica struct {
	ID        string `header:"id" json:"id"`
	Name      string `header:"name" json:"name"`
	State     string `header:"state" json:"state"`
	Region    string `header:"region" json:"region"`
	Cluster   string `header:"cluster" json:"cluster"`
	Replicas  int    `header:"replicas" json:"replicas"`
	Host      string `header:"host" json:"host"`
	Ready     bool   `header:"ready" json:"ready"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.ReadOnlyReplica
}

func (r *ReadOnlyReplica) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

type ReadOnlyReplicaChange struct {
	ID               string `header:"id" json:"id"`
	Replica          string `header:"replica,n/a" json:"replica"`
	State            string `header:"state" json:"state"`
	Cluster          string `header:"cluster" json:"cluster"`
	PreviousCluster  string `header:"previous cluster" json:"previous_cluster"`
	Replicas         int    `header:"replicas" json:"replicas"`
	PreviousReplicas int    `header:"previous replicas" json:"previous_replicas"`
	CreatedAt        int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	CompletedAt      *int64 `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`

	orig *ps.ReadOnlyReplicaChangeRequest
}

func (r *ReadOnlyReplicaChange) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func toReadOnlyReplica(replica *ps.ReadOnlyReplica) *ReadOnlyReplica {
	return &ReadOnlyReplica{
		ID:        replica.ID,
		Name:      replica.Name,
		State:     replica.State,
		Region:    replica.Region.Slug,
		Cluster:   replica.ClusterDisplayName,
		Replicas:  replica.Replicas,
		Host:      replica.AccessHostURL,
		Ready:     replica.Ready,
		CreatedAt: printer.GetMilliseconds(replica.CreatedAt),
		UpdatedAt: printer.GetMilliseconds(replica.UpdatedAt),
		orig:      replica,
	}
}

func toReadOnlyReplicas(replicas []*ps.ReadOnlyReplica) []*ReadOnlyReplica {
	result := make([]*ReadOnlyReplica, 0, len(replicas))
	for _, replica := range replicas {
		result = append(result, toReadOnlyReplica(replica))
	}
	return result
}

func toReadOnlyReplicaChange(change *ps.ReadOnlyReplicaChangeRequest) *ReadOnlyReplicaChange {
	var replica string
	if change.Replica != nil {
		replica = change.Replica.Name
	}
	return &ReadOnlyReplicaChange{
		ID:               change.ID,
		Replica:          replica,
		State:            change.State,
		Cluster:          change.ClusterDisplayName,
		PreviousCluster:  change.PreviousClusterDisplayName,
		Replicas:         change.Replicas,
		PreviousReplicas: change.PreviousReplicas,
		CreatedAt:        printer.GetMilliseconds(change.CreatedAt),
		CompletedAt:      printer.GetMillisecondsIfExists(change.CompletedAt),
		orig:             change,
	}
}

func toReadOnlyReplicaChanges(changes []*ps.ReadOnlyReplicaChangeRequest) []*ReadOnlyReplicaChange {
	result := make([]*ReadOnlyReplicaChange, 0, len(changes))
	for _, change := range changes {
		result = append(result, toReadOnlyReplicaChange(change))
	}
	return result
}
