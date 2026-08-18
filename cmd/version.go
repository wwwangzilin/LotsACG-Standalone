package cmd

import (
	"fmt"
	"runtime"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/version"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version number of LotsACG",
	Run: func(cmd *cobra.Command, args []string) {
		ShowVersion()
	},
}

func init() {
	rootCmd.AddCommand(VersionCmd)
}

func ShowVersion() {
	fmt.Printf("LotsACG version: %s %s/%s\nBuildTime: %s, Commit: %s\n",
		version.Version, runtime.GOOS, runtime.GOARCH, version.BuildTime, version.Commit)
}
