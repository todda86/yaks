package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "yaks",
	Short: "A Kubernetes context & namespace switcher",
	Long: `yaks (Yet Another Kontext Switcher) is a multiplatform Kubernetes
context and namespace switcher. It spawns isolated sub-shells with
per-session kubeconfig files so context changes don't leak between terminals.

Features:
  - Interactive context switching with fzf support
  - Namespace switching within a context
  - Isolated sub-shells per context/namespace
  - Shell prompt integration (bash, zsh, fish)
  - Nested shell support with depth tracking
  - Multi-kubeconfig file merging`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(ctxCmd)
	rootCmd.AddCommand(nsCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(initShellCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
}
