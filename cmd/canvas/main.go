package main

import (
	"os"

	"github.com/thedavidweng/canvas-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(cli.Version))
}
