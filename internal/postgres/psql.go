package postgres

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	exec "golang.org/x/sys/execabs"
)

var psqlVersionRegex = regexp.MustCompile(`psql \(PostgreSQL\) (\d+)\.?(\d*)`)

// FindPsqlPath locates a PostgreSQL psql client on PATH.
func FindPsqlPath() (string, error) {
	for _, cmd := range []string{"psql-18", "psql-17", "psql-16", "psql-15", "psql"} {
		path, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}
		c := exec.Command(path, "--version")
		out, err := c.Output()
		if err != nil {
			continue
		}
		if strings.Contains(string(out), "PostgreSQL") {
			return path, nil
		}
	}

	return "", fmt.Errorf("couldn't find the 'psql' command-line tool required for PostgreSQL imports.\n" +
		"To install, run: brew install postgresql@18")
}

// CheckPsqlVersion verifies psql meets a minimum major version.
func CheckPsqlVersion(minMajor int) (major, minor int, err error) {
	path, err := FindPsqlPath()
	if err != nil {
		return 0, 0, err
	}

	c := exec.Command(path, "--version")
	out, err := c.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get psql version: %w", err)
	}

	matches := psqlVersionRegex.FindStringSubmatch(string(out))
	if len(matches) < 2 {
		return 0, 0, fmt.Errorf("could not parse psql version from: %s", string(out))
	}

	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse psql major version: %w", err)
	}

	if len(matches) > 2 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}

	if major < minMajor {
		return major, minor, fmt.Errorf("psql version %d.%d is too old, minimum required is %d", major, minor, minMajor)
	}

	return major, minor, nil
}
