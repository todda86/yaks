package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultConfigPath_XDG(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping XDG test on windows")
	}

	t.Setenv("YAKS_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")

	got := DefaultConfigPath()
	want := "/custom/xdg/yaks/config.yaml"
	if got != want {
		t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

func TestDefaultConfigPath_YAKSConfig(t *testing.T) {
	t.Setenv("YAKS_CONFIG", "/my/custom/config.yaml")

	got := DefaultConfigPath()
	if got != "/my/custom/config.yaml" {
		t.Errorf("DefaultConfigPath() = %q, want /my/custom/config.yaml", got)
	}
}

func TestDefaultConfigPath_Home(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping home test on windows")
	}

	t.Setenv("YAKS_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := DefaultConfigPath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "yaks", "config.yaml")
	if got != want {
		t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	cfg, err := LoadConfigFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfigFrom() error: %v", err)
	}
	// Should return empty config, not an error
	if len(cfg.Hooks.Pre) != 0 || len(cfg.Hooks.Post) != 0 || len(cfg.Hooks.Exit) != 0 {
		t.Error("expected empty hooks from nonexistent config file")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `hooks:
  pre:
    - name: "warn-prod"
      match: "prod-*"
      command: "echo WARNING: production context!"
    - name: "global-pre"
      command: "echo entering context"
  post:
    - name: "set-bg"
      match: "prod-*"
      command: "set_terminal_bg red"
  exit:
    - name: "reset-bg"
      command: "reset_terminal_bg"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom() error: %v", err)
	}

	if len(cfg.Hooks.Pre) != 2 {
		t.Fatalf("len(Pre) = %d, want 2", len(cfg.Hooks.Pre))
	}
	if cfg.Hooks.Pre[0].Name != "warn-prod" {
		t.Errorf("Pre[0].Name = %q, want %q", cfg.Hooks.Pre[0].Name, "warn-prod")
	}
	if cfg.Hooks.Pre[0].Match != "prod-*" {
		t.Errorf("Pre[0].Match = %q, want %q", cfg.Hooks.Pre[0].Match, "prod-*")
	}
	if cfg.Hooks.Pre[1].Match != "" {
		t.Errorf("Pre[1].Match = %q, want empty (global hook)", cfg.Hooks.Pre[1].Match)
	}

	if len(cfg.Hooks.Post) != 1 {
		t.Fatalf("len(Post) = %d, want 1", len(cfg.Hooks.Post))
	}
	if cfg.Hooks.Post[0].Name != "set-bg" {
		t.Errorf("Post[0].Name = %q, want %q", cfg.Hooks.Post[0].Name, "set-bg")
	}

	if len(cfg.Hooks.Exit) != 1 {
		t.Fatalf("len(Exit) = %d, want 1", len(cfg.Hooks.Exit))
	}
	if cfg.Hooks.Exit[0].Name != "reset-bg" {
		t.Errorf("Exit[0].Name = %q, want %q", cfg.Hooks.Exit[0].Name, "reset-bg")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadConfigFrom(path)
	if err == nil {
		t.Fatal("LoadConfigFrom() expected error for invalid YAML, got nil")
	}
}

func TestMatchingHooks_ExactMatch(t *testing.T) {
	hooks := []Hook{
		{Name: "prod-only", Match: "prod-*", Command: "echo prod"},
		{Name: "staging", Match: "staging-*", Command: "echo staging"},
		{Name: "global", Match: "", Command: "echo all"},
	}

	matched := MatchingHooks(hooks, "prod-us-east")
	if len(matched) != 2 {
		t.Fatalf("len(matched) = %d, want 2", len(matched))
	}
	if matched[0].Name != "prod-only" {
		t.Errorf("matched[0].Name = %q, want %q", matched[0].Name, "prod-only")
	}
	if matched[1].Name != "global" {
		t.Errorf("matched[1].Name = %q, want %q", matched[1].Name, "global")
	}
}

func TestMatchingHooks_NoMatch(t *testing.T) {
	hooks := []Hook{
		{Name: "prod-only", Match: "prod-*", Command: "echo prod"},
	}

	matched := MatchingHooks(hooks, "dev-cluster")
	if len(matched) != 0 {
		t.Errorf("len(matched) = %d, want 0", len(matched))
	}
}

func TestMatchingHooks_GlobalOnly(t *testing.T) {
	hooks := []Hook{
		{Name: "global", Match: "", Command: "echo all"},
	}

	matched := MatchingHooks(hooks, "anything")
	if len(matched) != 1 {
		t.Fatalf("len(matched) = %d, want 1", len(matched))
	}
	if matched[0].Name != "global" {
		t.Errorf("matched[0].Name = %q, want %q", matched[0].Name, "global")
	}
}

func TestMatchingHooks_MultiplePatterns(t *testing.T) {
	hooks := []Hook{
		{Name: "eu", Match: "*-eu-*", Command: "echo eu"},
		{Name: "prod", Match: "prod-*", Command: "echo prod"},
		{Name: "all", Match: "", Command: "echo all"},
	}

	matched := MatchingHooks(hooks, "prod-eu-west")
	if len(matched) != 3 {
		t.Fatalf("len(matched) = %d, want 3", len(matched))
	}
}

func TestMatchingHooks_EmptyList(t *testing.T) {
	matched := MatchingHooks(nil, "anything")
	if len(matched) != 0 {
		t.Errorf("len(matched) = %d, want 0", len(matched))
	}
}

func TestRunHooks_ExecutesCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping hook execution test on windows")
	}

	dir := t.TempDir()
	marker := filepath.Join(dir, "hook-ran")

	hooks := []Hook{
		{Name: "test", Command: "touch " + marker},
	}

	RunHooks(hooks, os.Environ())

	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("hook command did not execute — marker file not created")
	}
}

func TestRunHooks_SkipsEmptyCommand(t *testing.T) {
	hooks := []Hook{
		{Name: "empty", Command: ""},
	}

	// Should not panic or error
	RunHooks(hooks, os.Environ())
}

func TestRunHooks_ContinuesAfterFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping hook execution test on windows")
	}

	dir := t.TempDir()
	marker := filepath.Join(dir, "second-ran")

	hooks := []Hook{
		{Name: "fail", Command: "false"},
		{Name: "succeed", Command: "touch " + marker},
	}

	RunHooks(hooks, os.Environ())

	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("second hook did not run after first hook failed")
	}
}

func TestRunHooks_InheritsEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping hook execution test on windows")
	}

	dir := t.TempDir()
	output := filepath.Join(dir, "env-output")

	env := append(os.Environ(),
		"YAKS_CONTEXT=test-ctx",
		"YAKS_NAMESPACE=test-ns",
	)

	hooks := []Hook{
		{Name: "check-env", Command: "echo $YAKS_CONTEXT:$YAKS_NAMESPACE > " + output},
	}

	RunHooks(hooks, env)

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != "test-ctx:test-ns" {
		t.Errorf("hook env output = %q, want %q", got, "test-ctx:test-ns")
	}
}

func TestParseContextFromEnv(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"YAKS_CONTEXT=my-ctx",
		"HOME=/home/user",
		"YAKS_NAMESPACE=my-ns",
	}

	ctx, ns := ParseContextFromEnv(env)
	if ctx != "my-ctx" {
		t.Errorf("context = %q, want %q", ctx, "my-ctx")
	}
	if ns != "my-ns" {
		t.Errorf("namespace = %q, want %q", ns, "my-ns")
	}
}

func TestParseContextFromEnv_Empty(t *testing.T) {
	ctx, ns := ParseContextFromEnv(nil)
	if ctx != "" || ns != "" {
		t.Errorf("expected empty strings, got context=%q namespace=%q", ctx, ns)
	}
}

func TestHookShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping unix shell test on windows")
	}

	t.Setenv("SHELL", "/bin/bash")
	got := hookShell()
	if got != "/bin/bash" {
		t.Errorf("hookShell() = %q, want /bin/bash", got)
	}
}
