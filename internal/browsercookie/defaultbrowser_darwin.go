//go:build darwin

package browsercookie

import (
	"os/exec"
	"strings"
)

// defaultBrowser returns the default browser identifier on macOS, or "" if unknown.
func defaultBrowser() string {
	out, err := exec.Command("defaults", "read",
		"com.apple.LaunchServices/com.apple.launchservices.secure",
		"LSHandlers").Output()
	if err != nil {
		return ""
	}
	// Look for the https handler entry.
	s := string(out)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "LSHandlerRoleAll") {
			switch {
			case strings.Contains(line, "com.google.chrome"):
				return "chrome"
			case strings.Contains(line, "com.apple.safari"):
				return "safari"
			case strings.Contains(line, "org.mozilla.firefox"):
				return "firefox"
			case strings.Contains(line, "com.microsoft.edgemac"):
				return "edge"
			case strings.Contains(line, "com.brave.browser"):
				return "brave"
			}
		}
	}
	return ""
}
