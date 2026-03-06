package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/todda86/yaks/pkg/fzf"
	"github.com/todda86/yaks/pkg/kubeconfig"
	"github.com/todda86/yaks/pkg/shell"
	"github.com/todda86/yaks/pkg/state"
)

var nsShellEval string

var nsCmd = &cobra.Command{
	Use:     "ns [namespace]",
	Aliases: []string{"namespace"},
	Short:   "Switch namespace in the current context",
	Long: `Switch the active namespace in the current context. If inside a yaks
shell, updates the isolated kubeconfig. Otherwise, updates the default
kubeconfig file.

If no namespace is given, an interactive selector is shown listing all
namespaces in the current cluster (requires kubectl access).`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeNamespaceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		var namespace string

		if len(args) > 0 {
			namespace = args[0]
			namespaces, err := getClusterNamespaces()
			if err != nil {
				return fmt.Errorf("failed to verify namespace: %w", err)
			}
			found := false
			for _, ns := range namespaces {
				if ns == namespace {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("namespace %q does not exist in the current cluster", namespace)
			}
		} else {
			namespaces, err := getClusterNamespaces()
			if err != nil {
				return fmt.Errorf("failed to list namespaces: %w\n(provide namespace as argument to skip auto-discovery)", err)
			}

			if len(namespaces) == 0 {
				return fmt.Errorf("no namespaces found")
			}

			currentNs := state.GetCurrentNamespace()
			items := make([]string, len(namespaces))
			for i, ns := range namespaces {
				if ns == currentNs {
					items[i] = fmt.Sprintf("%s (current)", ns)
				} else {
					items[i] = ns
				}
			}

			selected, err := fzf.Select(items, "Select namespace")
			if err != nil {
				return err
			}

			namespace = selected
			if len(namespace) > 10 && namespace[len(namespace)-10:] == " (current)" {
				namespace = namespace[:len(namespace)-10]
			}
		}

		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = kubeconfig.DefaultKubeconfigPath()
		}

		cfg, err := kubeconfig.Load(kubeconfigPath)
		if err != nil {
			return err
		}

		contextName := cfg.CurrentContext
		if contextName == "" {
			return fmt.Errorf("no current context set")
		}

		if err := cfg.SetNamespace(contextName, namespace); err != nil {
			return err
		}

		if err := kubeconfig.Save(cfg, kubeconfigPath); err != nil {
			return err
		}

		if state.IsActive() {
			os.Setenv("YAKS_NAMESPACE", namespace)
		}

		// Eval mode: output shell commands for the wrapper to eval
		if nsShellEval != "" {
			script := shell.NsEnvScript(nsShellEval, namespace)
			if script == "" {
				return fmt.Errorf("unsupported shell type for --shell-eval: %s", nsShellEval)
			}
			if !state.Quiet() {
				fmt.Fprintf(os.Stderr, "\033[1;33m%s\033[0m — namespace set\n", namespace)
			}
			fmt.Print(script)
			return nil
		}

		if !state.Quiet() {
			yellow := color.New(color.FgYellow, color.Bold)
			yellow.Printf("Namespace set to: %s\n", namespace)
		}

		return nil
	},
}

// completeNamespaceNames provides tab-completion for namespace names by
// querying the cluster via kubectl.
func completeNamespaceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ns, err := getClusterNamespaces()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return ns, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	nsCmd.Flags().StringVar(&nsShellEval, "shell-eval", "", "Output eval commands for namespace update (used by shell wrapper)")
	nsCmd.Flags().MarkHidden("shell-eval")
}

func getClusterNamespaces() ([]string, error) {
	c := exec.Command("kubectl", "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")
	out, err := c.Output()
	if err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	return strings.Fields(raw), nil
}
