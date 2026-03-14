package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/todda86/yaks/pkg/kubeconfig"
	"github.com/todda86/yaks/pkg/state"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show current context, namespace, and cluster info",
	Long: `Display information about the current Kubernetes context including
the context name, namespace, cluster server, and yaks shell depth.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cyan := color.New(color.FgCyan, color.Bold)
		yellow := color.New(color.FgYellow, color.Bold)
		green := color.New(color.FgGreen, color.Bold)
		white := color.New(color.Bold)
		dim := color.New(color.Faint)

		// Load the current (possibly isolated) kubeconfig for context details
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

		ctx, err := cfg.GetContext(contextName)
		if err != nil {
			return err
		}

		namespace := ctx.Context.Namespace
		if namespace == "" {
			namespace = "default"
		}

		fmt.Println()
		white.Println("+-----------------------------------------+")
		white.Println("|           yaks status                |")
		white.Println("+-----------------------------------------+")

		fmt.Printf("  Context:   ")
		cyan.Println(contextName)

		fmt.Printf("  Namespace: ")
		yellow.Println(namespace)

		fmt.Printf("  Cluster:   ")
		green.Println(ctx.Context.Cluster)

		cluster, err := cfg.GetCluster(ctx.Context.Cluster)
		if err == nil {
			fmt.Printf("  Server:    ")
			dim.Println(cluster.Cluster.Server)
		}

		fmt.Printf("  User:      ")
		dim.Println(ctx.Context.User)

		if state.IsActive() {
			fmt.Printf("  Shell:     ")
			green.Printf("active (depth: %d)\n", state.GetDepth())

			fmt.Printf("  Config:    ")
			dim.Println(kubeconfigPath)
		} else {
			fmt.Printf("  Shell:     ")
			dim.Println("not in yaks shell")
		}

		// Load all contexts (from YAKS_KUBECONFIG if inside a yaks session)
		// so the "Available contexts" list shows everything, not just the
		// isolated single-context temp file.
		allCfg, _, allErr := kubeconfig.LoadAll()
		if allErr != nil {
			allCfg = cfg // fall back to current config
		}

		fmt.Println()
		white.Println("  Available contexts:")
		for _, c := range allCfg.ListContextNames() {
			marker := "  "
			if c == contextName {
				marker = "> "
			}
			ns := ""
			for _, nc := range allCfg.Contexts {
				if nc.Name == c && nc.Context.Namespace != "" {
					ns = nc.Context.Namespace
				}
			}
			if c == contextName {
				cyan.Printf("    %s%s", marker, c)
			} else {
				fmt.Printf("    %s%s", marker, c)
			}
			if ns != "" {
				dim.Printf(" (%s)", ns)
			}
			fmt.Println()
		}

		fmt.Println()
		return nil
	},
}
