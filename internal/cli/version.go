package cli

import (
	"fmt"

	"github.com/forjd/aid/internal/output"
)

// BuildInfo describes the build identity populated by main via ldflags.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

var build BuildInfo

// SetBuildInfo records build metadata for the `aid version` command.
func SetBuildInfo(info BuildInfo) {
	build = info
}

func versionCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return newError(ErrCodeUsage, "version does not accept arguments")
	}

	version := build.Version
	if version == "" {
		version = "dev"
	}

	if streams.Options.IsJSON() {
		return output.WriteVersion(streams.Out, output.VersionResult{
			Version: version,
			Commit:  build.Commit,
			Date:    build.Date,
		})
	}

	fmt.Fprintf(streams.Out, "aid %s\n", version)
	if build.Commit != "" {
		fmt.Fprintf(streams.Out, "commit: %s\n", build.Commit)
	}
	if build.Date != "" {
		fmt.Fprintf(streams.Out, "built:  %s\n", build.Date)
	}
	return nil
}
