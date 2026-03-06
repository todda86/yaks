package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/todda86/yaks/pkg/prompt"
)

var initShellCmd = &cobra.Command{
	Use:   "init [bash|zsh|fish|powershell]",
	Short: "Print shell integration script",
	Long: `Print a shell script that integrates yaks into your prompt.

Add the following to your shell configuration:

  Bash (~/.bashrc):
    eval "$(yaks init bash)"

  Zsh (~/.zshrc):
    eval "$(yaks init zsh)"

  Fish (~/.config/fish/config.fish):
    yaks init fish | source

  PowerShell ($PROFILE):
    yaks init powershell | Out-String | Invoke-Expression`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		shellType := args[0]
		script := prompt.ShellInit(shellType)
		if script == "" {
			return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shellType)
		}
		fmt.Print(script)
		return nil
	},
}
