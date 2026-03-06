package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Hook defines a single hook action.
type Hook struct {
	Name    string `yaml:"name"`
	Match   string `yaml:"match"`
	Command string `yaml:"command"`
	Stop    bool   `yaml:"stop"`
}

// Config holds the hooks configuration.
type Config struct {
	Hooks HooksConfig `yaml:"hooks"`
}

// HooksConfig groups hooks by lifecycle phase.
type HooksConfig struct {
	Pre  []Hook `yaml:"pre"`
	Post []Hook `yaml:"post"`
	Exit []Hook `yaml:"exit"`
}

// DefaultConfigPath returns the platform-appropriate config file location.
func DefaultConfigPath() string {
	if p := os.Getenv("YAKS_CONFIG"); p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "yaks", "config.yaml")
		}
		return filepath.Join(os.Getenv("USERPROFILE"), ".config", "yaks", "config.yaml")
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "yaks", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "yaks", "config.yaml")
}

// LoadConfig reads and parses the config file. Returns empty Config if missing.
func LoadConfig() (*Config, error) {
	return LoadConfigFrom(DefaultConfigPath())
}

// LoadConfigFrom reads and parses config from a specific path.
func LoadConfigFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// MatchingHooks returns hooks whose Match pattern matches the context name.
// If a matched hook has Stop set to true, no further hooks are evaluated.
func MatchingHooks(hooks []Hook, contextName string) []Hook {
	var matched []Hook
	for _, h := range hooks {
		if h.Match == "" {
			matched = append(matched, h)
			if h.Stop {
				break
			}
			continue
		}
		if ok, _ := filepath.Match(h.Match, contextName); ok {
			matched = append(matched, h)
			if h.Stop {
				break
			}
		}
	}
	return matched
}

// RunHooks executes each hook command through the user shell.
// Failures print a warning but do not abort.
func RunHooks(hooks []Hook, env []string) {
	runHooksWithStdout(hooks, env, os.Stdout)
}

// RunHooksToStderr is like RunHooks but redirects hook stdout to stderr.
// Use this when the caller's stdout is reserved for structured output (e.g. eval scripts).
func RunHooksToStderr(hooks []Hook, env []string) {
	runHooksWithStdout(hooks, env, os.Stderr)
}

func runHooksWithStdout(hooks []Hook, env []string, stdout *os.File) {
	shellBin := hookShell()
	for _, h := range hooks {
		if h.Command == "" {
			continue
		}
		cmd := exec.Command(shellBin, "-c", h.Command)
		cmd.Env = env
		cmd.Stdin = os.Stdin
		cmd.Stdout = stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			label := h.Name
			if label == "" {
				label = h.Command
			}
			fmt.Fprintf(os.Stderr, "yaks: hook %q failed: %v\n", label, err)
		}
	}
}

func hookShell() string {
	if runtime.GOOS == "windows" {
		if ps, err := exec.LookPath("pwsh.exe"); err == nil {
			return ps
		}
		if ps, err := exec.LookPath("powershell.exe"); err == nil {
			return ps
		}
		return "cmd.exe"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	for _, s := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(s); err == nil {
			return s
		}
	}
	return "/bin/sh"
}

// ParseContextFromEnv extracts context and namespace from an env slice.
func ParseContextFromEnv(env []string) (context, namespace string) {
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "YAKS_CONTEXT":
			context = parts[1]
		case "YAKS_NAMESPACE":
			namespace = parts[1]
		}
	}
	return
}
