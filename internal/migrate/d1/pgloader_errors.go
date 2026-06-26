package d1

import (
	"regexp"
	"strings"
)

var pgloaderSummaryErrorRe = regexp.MustCompile(`(?m)^\|\s+(\d+)\s+\|`)

// pgloaderHadErrors inspects pgloader output for failures that do not set exit code.
func pgloaderHadErrors(output string) bool {
	if strings.Contains(output, "Database error") ||
		strings.Contains(output, "INSUFFICIENT-PRIVILEGE") ||
		strings.Contains(output, "must be owner of table") {
		return true
	}
	for _, match := range pgloaderSummaryErrorRe.FindAllStringSubmatch(output, -1) {
		if len(match) < 2 {
			continue
		}
		if match[1] != "0" {
			return true
		}
	}
	return false
}
