package printer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lensesio/tableprinter"
)

// PrintOutput prints the output as JSON or in a table format.
func PrintOutput(isJSON bool, obj interface{}) error {
	if isJSON {
		output, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}

		fmt.Print(string(output))
	} else {
		tableprinter.Print(os.Stdout, obj)
	}

	return nil
}
