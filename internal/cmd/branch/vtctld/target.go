package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/printer"
)

func progressTarget(organization, database, branch string) string {
	return fmt.Sprintf("%s/%s/%s",
		printer.BoldBlue(organization),
		printer.BoldBlue(database),
		printer.BoldBlue(branch))
}
