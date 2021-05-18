package cmdutil

import (
	"os"
	"strings"

	"github.com/planetscale/cli/internal/printer"

	exec "golang.org/x/sys/execabs"
)

const ApplicationURL = "https://app.planetscale.com"

// OpenBrowser opens a web browser at the specified url.
func OpenBrowser(goos, url string) *exec.Cmd {
	if !printer.IsTTY {
		panic("OpenBrowser called without a TTY")
	}

	exe := "open"
	var args []string
	switch goos {
	case "darwin":
		args = append(args, url)
	case "windows":
		exe, _ = exec.LookPath("cmd")
		r := strings.NewReplacer("&", "^&")
		args = append(args, "/c", "start", r.Replace(url))
	default:
		exe = linuxExe()
		args = append(args, url)
	}

	cmd := exec.Command(exe, args...)
	cmd.Stderr = os.Stderr
	return cmd
}

func linuxExe() string {
	exe := "xdg-open"

	_, err := exec.LookPath(exe)
	if err != nil {
		_, err := exec.LookPath("wslview")
		if err == nil {
			exe = "wslview"
		}
	}

	return exe
}
