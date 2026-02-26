package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/todda86/yaks/pkg/hooks"
	"github.com/todda86/yaks/pkg/kubeconfig"
	"github.com/todda86/yaks/pkg/state"
)

// SpawnShell spawns a new sub-shell with the given context and namespace isolated
// via a temporary kubeconfig file. Pre/post/exit hooks from the config file are
// executed at the appropriate lifecycle points.
func SpawnShell(contextName, namespace string) error {
	return SpawnShellWithConfig(contextName, namespace, nil)
}

// SpawnShellWithConfig is like SpawnShell but accepts an explicit hooks config.
// Pass nil to load the default config file.
func SpawnShellWithConfig(contextName, namespace string, cfg *hooks.Config) error {
	tmpDir, tmpKubeconfig, ctx, env, err := setupIsolatedEnv(contextName, namespace)
	if err != nil {
		return err
	}

	depth := state.GetDepth() + 1
	env = buildEnv(tmpKubeconfig, contextName, ctx.Context.Namespace, depth)

	// Load hooks config
	if cfg == nil {
		cfg, err = hooks.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "yaks: warning: failed to load hooks config: %v\n", err)
			cfg = &hooks.Config{}
		}
	}

	// Run pre-switch hooks (before spawning the shell)
	preHooks := hooks.MatchingHooks(cfg.Hooks.Pre, contextName)
	if len(preHooks) > 0 {
		hooks.RunHooks(preHooks, env)
	}

	// Detect shell
	shellBin := detectShell()

	// Spawn the shell
	cmd := exec.Command(shellBin)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if !state.Quiet() {
		fmt.Printf("\033[1;36m%s\033[0m|\033[1;33m%s\033[0m (depth: %d) — type 'exit' to return\n",
			contextName, ctx.Context.Namespace, depth)
	}

	// Run post-switch hooks (shell is about to start, terminal is set up)
	postHooks := hooks.MatchingHooks(cfg.Hooks.Post, contextName)
	if len(postHooks) > 0 {
		hooks.RunHooks(postHooks, env)
	}

	if err := cmd.Run(); err != nil {
		// Non-zero exit from user shell is not an error for us
		if exitErr, ok := err.(*exec.ExitError); ok {
			_ = exitErr
		} else {
			return fmt.Errorf("failed to spawn shell: %w", err)
		}
	}

	// Run exit hooks (shell has exited, clean up)
	exitHooks := hooks.MatchingHooks(cfg.Hooks.Exit, contextName)
	if len(exitHooks) > 0 {
		hooks.RunHooks(exitHooks, env)
	}

	// Clean up temp files
	os.RemoveAll(tmpDir)

	return nil
}

// ExecCommand runs a command in the given context/namespace without spawning an
// interactive shell. The command inherits stdin/stdout/stderr and its exit code
// is returned. Pre and exit hooks are executed around the command.
func ExecCommand(contextName, namespace string, command []string) (int, error) {
	return ExecCommandWithConfig(contextName, namespace, command, nil)
}

// ExecCommandWithConfig is like ExecCommand but accepts an explicit hooks config.
func ExecCommandWithConfig(contextName, namespace string, command []string, cfg *hooks.Config) (int, error) {
	tmpDir, tmpKubeconfig, ctx, _, err := setupIsolatedEnv(contextName, namespace)
	if err != nil {
		return 1, err
	}
	defer os.RemoveAll(tmpDir)

	env := buildEnv(tmpKubeconfig, contextName, ctx.Context.Namespace, state.GetDepth()+1)

	// Load hooks config
	if cfg == nil {
		cfg, err = hooks.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "yaks: warning: failed to load hooks config: %v\n", err)
			cfg = &hooks.Config{}
		}
	}

	// Run pre hooks
	preHooks := hooks.MatchingHooks(cfg.Hooks.Pre, contextName)
	if len(preHooks) > 0 {
		hooks.RunHooks(preHooks, env)
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	runErr := cmd.Run()

	// Run exit hooks
	exitHooks := hooks.MatchingHooks(cfg.Hooks.Exit, contextName)
	if len(exitHooks) > 0 {
		hooks.RunHooks(exitHooks, env)
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("failed to exec command: %w", runErr)
	}

	return 0, nil
}

// setupIsolatedEnv prepares a temporary kubeconfig scoped to one context/namespace
// and returns the tmpDir path, kubeconfig path, resolved context, and base env.
func setupIsolatedEnv(contextName, namespace string) (tmpDir, tmpKubeconfig string, ctx *kubeconfig.NamedContext, env []string, err error) {
	// Load the full kubeconfig
	cfg, _, err := kubeconfig.LoadAll()
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Verify context exists
	ctx, err = cfg.GetContext(contextName)
	if err != nil {
		return "", "", nil, nil, err
	}

	// Set the namespace on the context
	if namespace != "" {
		ctx.Context.Namespace = namespace
	} else if ctx.Context.Namespace == "" {
		ctx.Context.Namespace = "default"
	}

	// Build a minimal kubeconfig with just this context
	isolatedConfig := buildIsolatedConfig(cfg, ctx)

	// Write to a temp file
	tmpDir, err = os.MkdirTemp("", "yaks-*")
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	tmpKubeconfig = filepath.Join(tmpDir, "config")
	if err := kubeconfig.Save(isolatedConfig, tmpKubeconfig); err != nil {
		os.RemoveAll(tmpDir)
		return "", "", nil, nil, fmt.Errorf("failed to write temp kubeconfig: %w", err)
	}

	depth := state.GetDepth() + 1
	env = buildEnv(tmpKubeconfig, contextName, ctx.Context.Namespace, depth)

	return tmpDir, tmpKubeconfig, ctx, env, nil
}

// buildIsolatedConfig creates a minimal kubeconfig with only the specified context.
func buildIsolatedConfig(full *kubeconfig.KubeConfig, namedCtx *kubeconfig.NamedContext) *kubeconfig.KubeConfig {
	iso := &kubeconfig.KubeConfig{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: namedCtx.Name,
	}

	// Add the context
	iso.Contexts = []kubeconfig.NamedContext{*namedCtx}

	// Find and add the referenced cluster
	for _, c := range full.Clusters {
		if c.Name == namedCtx.Context.Cluster {
			iso.Clusters = append(iso.Clusters, c)
			break
		}
	}

	// Find and add the referenced user
	for _, u := range full.Users {
		if u.Name == namedCtx.Context.User {
			iso.Users = append(iso.Users, u)
			break
		}
	}

	return iso
}

// detectShell detects the user's preferred shell.
func detectShell() string {
	if runtime.GOOS == "windows" {
		if ps, err := exec.LookPath("pwsh.exe"); err == nil {
			return ps
		}
		if ps, err := exec.LookPath("powershell.exe"); err == nil {
			return ps
		}
		return "cmd.exe"
	}

	// Unix: check SHELL env var
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// Fallback
	for _, s := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(s); err == nil {
			return s
		}
	}

	return "/bin/sh"
}

// buildEnv creates the environment for the subshell with yaks-specific variables.
func buildEnv(kubeconfigPath, context, namespace string, depth int) []string {
	env := os.Environ()

	// Filter out existing KUBECONFIG and yaks vars
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		key := strings.SplitN(e, "=", 2)[0]
		switch key {
		case "KUBECONFIG", "YAKS_CONTEXT", "YAKS_NAMESPACE", "YAKS_DEPTH", "YAKS_ACTIVE":
			continue
		default:
			filtered = append(filtered, e)
		}
	}

	// Add yaks-specific vars
	filtered = append(filtered,
		fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath),
		fmt.Sprintf("YAKS_CONTEXT=%s", context),
		fmt.Sprintf("YAKS_NAMESPACE=%s", namespace),
		fmt.Sprintf("YAKS_DEPTH=%d", depth),
		"YAKS_ACTIVE=1",
	)

	return filtered
}
