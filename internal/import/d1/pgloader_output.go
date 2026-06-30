package d1

import (
	"fmt"
	"regexp"
	"strconv"
)

const (
	pgloaderNoRowsRemediation = "Check pgloader stderr for table filter or cast errors; re-run import d1 start after fixing the dump or CLI"
)

var pgloaderFetchMetaDataRe = regexp.MustCompile(`(?m)^\s*fetch meta data\s+\d+\s+(\d+)`)

// pgloaderFetchMetaDataTableCount returns how many source tables pgloader matched, or -1 if absent.
func pgloaderFetchMetaDataTableCount(output string) int {
	matches := pgloaderFetchMetaDataRe.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return -1
	}
	n, err := strconv.Atoi(matches[len(matches)-1][1])
	if err != nil {
		return -1
	}
	return n
}

// pgloaderRowsCopied parses the pgloader report summary row count for table, if present.
func pgloaderRowsCopied(output, table string) (int64, bool) {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(table) + `\s+\d+\s+(\d+)\s+`)
	m := re.FindStringSubmatch(output)
	if len(m) < 2 {
		return 0, false
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func validatePgloaderTableLoad(output, table string, expectedRows int) error {
	if expectedRows <= 0 {
		return nil
	}

	metaCount := pgloaderFetchMetaDataTableCount(output)
	if metaCount == 0 {
		return newMigrationError(
			ErrCodeImportFailed,
			fmt.Sprintf("pgloader matched 0 source tables for %q (expected %d rows)", table, expectedRows),
			pgloaderNoRowsRemediation,
		)
	}

	rows, found := pgloaderRowsCopied(output, table)
	if !found || rows == 0 {
		msg := fmt.Sprintf("pgloader copied 0 rows into %q (expected %d from dump)", table, expectedRows)
		if metaCount > 0 {
			msg = fmt.Sprintf("pgloader copied 0 rows into %q (expected %d from dump; matched %d source table(s))", table, expectedRows, metaCount)
		}
		return newMigrationError(ErrCodeImportFailed, msg, pgloaderNoRowsRemediation)
	}
	if rows != int64(expectedRows) {
		return newMigrationError(
			ErrCodeImportFailed,
			fmt.Sprintf("pgloader copied %d rows into %q (expected %d from dump)", rows, table, expectedRows),
			pgloaderNoRowsRemediation,
		)
	}

	return nil
}
