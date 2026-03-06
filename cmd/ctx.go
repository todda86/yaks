package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/todda86/yaks/pkg/fzf"
	"github.com/todda86/yaks/pkg/hooks"
	"github.com/todda86/yaks/pkg/kubeconfig"
	"github.com/todda86/yaks/pkg/shell"
	"github.com/todda86/yaks/pkg/state"
)

var ctxNamespace string
var ctxShellEval string

var ctxCmd = &cobra.Command{
	Use:     "ctx [context-name]",
	Aliases: []string{"context"},
	Short:   "Switch to a Kubernetes context in a new shell",
	Long: `Switch to a Kubernetes context by spawning a new sub-shell with an
isolated kubeconfig. If no context name is given, an interactive
selector is shown (using fzf if available).

The sub-shell gets its own KUBECONFIG pointing to a temporary file
containing only the selected context, so changes don't affect other
terminals.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeContextNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := kubeconfig.LoadAll()
		if err != nil {
			return err
		}

		var contextName string

		if len(args) > 0 {
			contextName = args[0]
			if _, err := cfg.GetContext(contextName); err != nil {
				return err
			}
		} else {
			contexts := cfg.ListContextNames()
			if len(contexts) == 0 {
				return fmt.Errorf("no contexts found in kubeconfig")
			}

			current := cfg.CurrentContext
			items := make([]string, len(contexts))
			for i, c := range contexts {
				if c == current {
					items[i] = fmt.Sprintf("%s (current)", c)
				} else {
					items[i] = c
				}
			}

			selected, err := fzf.Select(items, "Select context")
			if err != nil {
				return err
			}

			contextName = selected
			if len(contextName) > 10 && contextName[len(contextName)-10:] == " (current)" {
				contextName = contextName[:len(contextName)-10]
			}
		}

		// Eval mode: output shell commands for the wrapper function to eval
		if ctxShellEval != "" {
			tmpDir, kubeconfigPath, _, resolvedNs, err := shell.SetupIsolatedEnv(contextName, ctxNamespace)
			if err != nil {
				return err
			}

			originalKC := shell.OriginalKubeconfig()

			// Build env for pre-hooks
			env := os.Environ()
			env = append(env,
				fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath),
				fmt.Sprintf("YAKS_CONTEXT=%s", contextName),
				fmt.Sprintf("YAKS_NAMESPACE=%s", resolvedNs),
				"YAKS_ACTIVE=1",
			)

			// Run pre-hooks (stdout→stderr so eval output stays clean)
			hooksCfg, herr := hooks.LoadConfig()
			if herr != nil {
				fmt.Fprintf(os.Stderr, "yaks: warning: failed to load hooks config: %v\n", herr)
			} else {
				preHooks := hooks.MatchingHooks(hooksCfg.Hooks.Pre, contextName)
				if len(preHooks) > 0 {
					hooks.RunHooksToStderr(preHooks, env)
				}
			}

			if !state.Quiet() {
				fmt.Fprintf(os.Stderr, "\033[1;36m%s\033[0m|\033[1;33m%s\033[0m — switched\n", contextName, resolvedNs)
			}

			script := shell.EnvScript(ctxShellEval, tmpDir, kubeconfigPath, originalKC, contextName, resolvedNs)
			if script == "" {
				os.RemoveAll(tmpDir)
				return fmt.Errorf("unsupported shell type for --shell-eval: %s", ctxShellEval)
			}

			fmt.Print(script)

			// Run post-hooks (stdout→stderr so eval output stays clean)
			if herr == nil {
				postHooks := hooks.MatchingHooks(hooksCfg.Hooks.Post, contextName)
				if len(postHooks) > 0 {
					hooks.RunHooksToStderr(postHooks, env)
				}
			}

			return nil
		}

		// Subshell mode (legacy / no shell wrapper)
		if !state.Quiet() {
			cyan := color.New(color.FgCyan, color.Bold)
			cyan.Printf("Switching to context: %s\n", contextName)
		}

		return shell.SpawnShell(contextName, ctxNamespace)
	},
}

// completeContextNames provides tab-completion for Kubernetes context names.
func completeContextNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, _, err := kubeconfig.LoadAll()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return cfg.ListContextNames(), cobra.ShellCompDirectiveNoFileComp
}

func init() {
	ctxCmd.Flags().StringVarP(&ctxNamespace, "namespace", "n", "", "Set the namespace for the new shell")
	ctxCmd.Flags().StringVar(&ctxShellEval, "shell-eval", "", "Output eval commands for the given shell type (used by shell wrapper)")
	ctxCmd.Flags().MarkHidden("shell-eval")
}
