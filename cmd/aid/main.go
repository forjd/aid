package main

import (
	"io"
	"os"

	"github.com/forjd/aid/internal/cli"
)

// Build-time metadata injected by GoReleaser via -X main.version etc.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// run invokes the CLI with the supplied args/streams. It is exposed for tests.
func run(args []string, stdout, stderr io.Writer) int {
	cli.SetBuildInfo(cli.BuildInfo{Version: version, Commit: commit, Date: date})
	return cli.Run(args, stdout, stderr)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
