package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/todda86/yaks/pkg/prompt"
)

var initModule bool
var initInstall bool

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
    yaks init powershell | Out-String | Invoke-Expression

For enterprise/module deployments on PowerShell, use --module to generate
a proper PowerShell module with FunctionsToExport:

  yaks init powershell --module            # prints module files to stdout
  yaks init powershell --module --install   # installs to the modules path`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		shellType := args[0]

		// --module is only valid for powershell
		if initModule && shellType != "powershell" {
			return fmt.Errorf("--module is only supported for powershell")
		}

		if initModule {
			return handlePowerShellModule()
		}

		script := prompt.ShellInit(shellType)
		if script == "" {
			return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", shellType)
		}
		fmt.Print(script)
		return nil
	},
}

func handlePowerShellModule() error {
	psd1 := prompt.PowerShellModuleManifest()
	psm1 := prompt.PowerShellModuleScript()

	if initInstall {
		installDir, err := psModuleInstallPath()
		if err != nil {
			return err
		}

		moduleDir := filepath.Join(installDir, "YaksInit")
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			return fmt.Errorf("failed to create module directory %s: %w", moduleDir, err)
		}

		psd1Path := filepath.Join(moduleDir, "YaksInit.psd1")
		if err := os.WriteFile(psd1Path, []byte(psd1), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", psd1Path, err)
		}

		psm1Path := filepath.Join(moduleDir, "YaksInit.psm1")
		if err := os.WriteFile(psm1Path, []byte(psm1), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", psm1Path, err)
		}

		fmt.Fprintf(os.Stderr, "Installed YaksInit module to %s\n", moduleDir)
		fmt.Fprintf(os.Stderr, "Add to your $PROFILE:\n  Import-Module YaksInit\n")
		return nil
	}

	// Print both files to stdout with markers for extraction
	fmt.Println("# ===== YaksInit.psd1 =====")
	fmt.Print(psd1)
	fmt.Println()
	fmt.Println("# ===== YaksInit.psm1 =====")
	fmt.Print(psm1)
	return nil
}

// psModuleInstallPath returns the best PowerShell modules directory for the current platform.
func psModuleInstallPath() (string, error) {
	// Prefer PSModulePath env var first segment for user-writable location
	if psModPath := os.Getenv("PSModulePath"); psModPath != "" {
		sep := ":"
		if runtime.GOOS == "windows" {
			sep = ";"
		}
		paths := filepath.SplitList(psModPath)
		if len(paths) == 0 {
			// SplitList didn't work with the separator, try manually
			for _, p := range splitString(psModPath, sep) {
				if p != "" {
					return p, nil
				}
			}
		} else {
			return paths[0], nil
		}
	}

	// Platform defaults
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "Documents", "PowerShell", "Modules"), nil
	default:
		return filepath.Join(home, ".local", "share", "powershell", "Modules"), nil
	}
}

func splitString(s, sep string) []string {
	var parts []string
	for len(s) > 0 {
		i := indexOf(s, sep)
		if i < 0 {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:i])
		s = s[i+len(sep):]
	}
	return parts
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func init() {
	initShellCmd.Flags().BoolVar(&initModule, "module", false, "Generate a PowerShell module (YaksInit.psd1 + YaksInit.psm1)")
	initShellCmd.Flags().BoolVar(&initInstall, "install", false, "Install the module to the PowerShell modules path (use with --module)")
}
