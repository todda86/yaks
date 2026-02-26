package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of yaks",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("yaks version %s\n", version)
	},
}
