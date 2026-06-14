//go:build !darwin

package browsercookie

// defaultBrowser returns "" on non-macOS platforms (detection not implemented).
func defaultBrowser() string {
	return ""
}
