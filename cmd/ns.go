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
	"github.com/todda86/yaks/pkg/state"
)

var nsCmd = &cobra.Command{
	Use:     "ns [namespace]",
	Aliases: []string{"namespace"},
	Short:   "Switch namespace in the current context",
	Long: `Switch the active namespace in the current context. If inside a yaks
shell, updates the isolated kubeconfig. Otherwise, updates the default
kubeconfig file.

If no namespace is given, an interactive selector is shown listing all
namespaces in the current cluster (requires kubectl access).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var namespace string

		if len(args) > 0 {
			namespace = args[0]
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

		if !state.Quiet() {
			yellow := color.New(color.FgYellow, color.Bold)
			yellow.Printf("Namespace set to: %s\n", namespace)
		}

		return nil
	},
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
