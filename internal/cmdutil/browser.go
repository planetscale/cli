package cmdutil

import (
	"os"
	"os/exec"
	"strings"

	"github.com/cli/safeexec"
)

const ApplicationURL = "https://app.planetscale.com"

// OpenBrowser opens a web browser at the specified url.
func OpenBrowser(goos, url string) *exec.Cmd {
	if !IsTTY {
		panic("OpenBrowser called without a TTY")
	}
	exe := "open"
	var args []string
	switch goos {
	case "darwin":
		args = append(args, url)
	case "windows":
		exe, _ = lookPath("cmd")
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

	_, err := lookPath(exe)
	if err != nil {
		_, err := lookPath("wslview")
		if err == nil {
			exe = "wslview"
		}
	}

	return exe
}

var lookPath = safeexec.LookPath
