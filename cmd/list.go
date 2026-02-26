package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/todda86/yaks/pkg/kubeconfig"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available contexts and namespaces",
	Long:  `List all available Kubernetes contexts from your kubeconfig files.`,
}

var listCtxCmd = &cobra.Command{
	Use:     "contexts",
	Aliases: []string{"ctx"},
	Short:   "List all available contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := kubeconfig.LoadAll()
		if err != nil {
			return err
		}

		cyan := color.New(color.FgCyan, color.Bold)
		yellow := color.New(color.FgYellow)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  \tCONTEXT\tCLUSTER\tNAMESPACE\tUSER")
		fmt.Fprintln(w, "  \t-------\t-------\t---------\t----")

		for _, ctx := range cfg.Contexts {
			marker := " "
			if ctx.Name == cfg.CurrentContext {
				marker = ">"
			}
			ns := ctx.Context.Namespace
			if ns == "" {
				ns = "default"
			}

			if ctx.Name == cfg.CurrentContext {
				fmt.Fprintf(w, "%s\t", marker)
				cyan.Fprintf(w, "%s", ctx.Name)
				fmt.Fprintf(w, "\t%s\t", ctx.Context.Cluster)
				yellow.Fprintf(w, "%s", ns)
				fmt.Fprintf(w, "\t%s\n", ctx.Context.User)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					marker, ctx.Name, ctx.Context.Cluster, ns, ctx.Context.User)
			}
		}
		w.Flush()
		return nil
	},
}

var listNsCmd = &cobra.Command{
	Use:     "namespaces",
	Aliases: []string{"ns"},
	Short:   "List all namespaces in the current cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespaces, err := getClusterNamespaces()
		if err != nil {
			return fmt.Errorf("failed to list namespaces: %w", err)
		}

		yellow := color.New(color.FgYellow, color.Bold)
		currentNs := os.Getenv("YAKS_NAMESPACE")
		if currentNs == "" {
			kubeconfigPath := os.Getenv("KUBECONFIG")
			if kubeconfigPath == "" {
				kubeconfigPath = kubeconfig.DefaultKubeconfigPath()
			}
			cfg, err := kubeconfig.Load(kubeconfigPath)
			if err == nil && cfg.CurrentContext != "" {
				ctx, err := cfg.GetContext(cfg.CurrentContext)
				if err == nil {
					currentNs = ctx.Context.Namespace
				}
			}
		}

		for _, ns := range namespaces {
			if ns == currentNs {
				yellow.Printf("> %s\n", ns)
			} else {
				fmt.Printf("  %s\n", ns)
			}
		}
		return nil
	},
}

func init() {
	listCmd.AddCommand(listCtxCmd)
	listCmd.AddCommand(listNsCmd)
}
