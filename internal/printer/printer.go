package printer

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lensesio/tableprinter"
)

// PrintOutput prints the output as JSON or in a table format.
func PrintOutput(isJSON bool, obj interface{}) error {
	if isJSON {
		return PrintJSON(obj)
	}

	tableprinter.Print(os.Stdout, obj)
	return nil
}

// PrintJSON pretty prints the object as JSON.
func PrintJSON(obj interface{}) error {
	output, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}

	fmt.Print(string(output))

	return nil
}

func getMilliseconds(timestamp time.Time) int64 {
	numSeconds := timestamp.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))

	return numSeconds
}

func getMillisecondsIfExists(timestamp *time.Time) *int64 {
	if timestamp == nil {
		return nil
	}

	numSeconds := getMilliseconds(*timestamp)

	return &numSeconds
}
