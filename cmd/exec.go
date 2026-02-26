package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/todda86/yaks/pkg/shell"
)

var execNamespace string

var execCmd = &cobra.Command{
	Use:   "exec <context> <namespace> -- <command> [args...]",
	Short: "Run a command in a context/namespace without a shell",
	Long: `Execute a command in the specified context and namespace without
spawning an interactive shell. The command runs with an isolated
kubeconfig and inherits stdin/stdout/stderr. The process exits
with the same exit code as the executed command.

Examples:
  yaks exec my-cluster default -- kubectl get pods
  yaks exec prod kube-system -- helm list
  yaks exec staging -n monitoring -- kubectl logs -f deploy/prometheus`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		contextName := args[0]

		var namespace string
		var command []string

		// Find -- separator in os.Args to split positional from command args
		dashIdx := -1
		for i, a := range os.Args {
			if a == "--" {
				dashIdx = i
				break
			}
		}

		if dashIdx >= 0 && dashIdx+1 < len(os.Args) {
			// -- was used: exec <context> [namespace] -- <cmd...>
			command = os.Args[dashIdx+1:]
			if execNamespace != "" {
				namespace = execNamespace
			} else if len(args) > 1 {
				namespace = args[1]
			}
		} else {
			// No --: exec <context> <namespace> <cmd...>
			if execNamespace != "" {
				namespace = execNamespace
				command = args[1:]
			} else if len(args) >= 3 {
				namespace = args[1]
				command = args[2:]
			} else {
				return cmd.Help()
			}
		}

		if len(command) == 0 {
			return cmd.Help()
		}

		exitCode, err := shell.ExecCommand(contextName, namespace, command)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

func init() {
	execCmd.Flags().StringVarP(&execNamespace, "namespace", "n", "", "Namespace to use (alternative to positional arg)")
}
