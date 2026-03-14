package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/todda86/yaks/pkg/hooks"
	"github.com/todda86/yaks/pkg/kubeconfig"
	"github.com/todda86/yaks/pkg/shell"
	"github.com/todda86/yaks/pkg/state"
)

var activateNamespace string
var activateShellEval string

var activateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate a yaks session for the current kubeconfig context",
	Long: `Activate a yaks session by reading the current context from your
kubeconfig and emitting shell-eval output. This removes the need to
shell out to kubectl at startup.

Usage in shell init:

  Bash (~/.bashrc):
    eval "$(yaks activate --shell-eval bash)"

  Zsh (~/.zshrc):
    eval "$(yaks activate --shell-eval zsh)"

  Fish (~/.config/fish/config.fish):
    yaks activate --shell-eval fish | source

  PowerShell ($PROFILE):
    yaks activate --shell-eval powershell | Out-String | Invoke-Expression`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if activateShellEval == "" {
			return fmt.Errorf("--shell-eval is required (bash, zsh, fish, or powershell)")
		}

		cfg, _, err := kubeconfig.LoadAll()
		if err != nil {
			return err
		}

		contextName := cfg.CurrentContext
		if contextName == "" {
			return fmt.Errorf("no current context set in kubeconfig")
		}

		tmpDir, kubeconfigPath, _, resolvedNs, err := shell.SetupIsolatedEnv(contextName, activateNamespace)
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
			fmt.Fprintf(os.Stderr, "\033[1;36m%s\033[0m|\033[1;33m%s\033[0m — activated\n", contextName, resolvedNs)
		}

		script := shell.EnvScript(activateShellEval, tmpDir, kubeconfigPath, originalKC, contextName, resolvedNs)
		if script == "" {
			os.RemoveAll(tmpDir)
			return fmt.Errorf("unsupported shell type for --shell-eval: %s", activateShellEval)
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
	},
}

func init() {
	activateCmd.Flags().StringVarP(&activateNamespace, "namespace", "n", "", "Override the namespace for the session")
	activateCmd.Flags().StringVar(&activateShellEval, "shell-eval", "", "Shell type to generate eval commands for (bash, zsh, fish, powershell)")
	activateCmd.MarkFlagRequired("shell-eval")
}
