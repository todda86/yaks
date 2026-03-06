package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/todda86/yaks/pkg/kubeconfig"
)

// writeSampleKubeconfig writes a test kubeconfig and returns its path.
func writeSampleKubeconfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := `apiVersion: v1
kind: Config
current-context: test-ctx
clusters:
- name: test-cluster
  cluster:
    server: https://localhost:6443
contexts:
- name: test-ctx
  context:
    cluster: test-cluster
    user: test-user
    namespace: default
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}
	return path
}

func TestBuildIsolatedConfig(t *testing.T) {
	full := &kubeconfig.KubeConfig{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: "ctx-a",
		Clusters: []kubeconfig.NamedCluster{
			{Name: "cluster-a", Cluster: kubeconfig.Cluster{Server: "https://a:6443"}},
			{Name: "cluster-b", Cluster: kubeconfig.Cluster{Server: "https://b:6443"}},
		},
		Contexts: []kubeconfig.NamedContext{
			{Name: "ctx-a", Context: kubeconfig.Context{Cluster: "cluster-a", User: "user-a", Namespace: "ns-a"}},
			{Name: "ctx-b", Context: kubeconfig.Context{Cluster: "cluster-b", User: "user-b", Namespace: "ns-b"}},
		},
		Users: []kubeconfig.NamedUser{
			{Name: "user-a", User: kubeconfig.User{Token: "token-a"}},
			{Name: "user-b", User: kubeconfig.User{Token: "token-b"}},
		},
	}

	namedCtx := &full.Contexts[0] // ctx-a
	iso := buildIsolatedConfig(full, namedCtx)

	if iso.CurrentContext != "ctx-a" {
		t.Errorf("CurrentContext = %q, want %q", iso.CurrentContext, "ctx-a")
	}
	if iso.APIVersion != "v1" {
		t.Errorf("APIVersion = %q, want %q", iso.APIVersion, "v1")
	}
	if iso.Kind != "Config" {
		t.Errorf("Kind = %q, want %q", iso.Kind, "Config")
	}

	// Should only have the one context
	if len(iso.Contexts) != 1 {
		t.Fatalf("len(Contexts) = %d, want 1", len(iso.Contexts))
	}
	if iso.Contexts[0].Name != "ctx-a" {
		t.Errorf("Context name = %q, want %q", iso.Contexts[0].Name, "ctx-a")
	}

	// Should only have cluster-a
	if len(iso.Clusters) != 1 {
		t.Fatalf("len(Clusters) = %d, want 1", len(iso.Clusters))
	}
	if iso.Clusters[0].Name != "cluster-a" {
		t.Errorf("Cluster name = %q, want %q", iso.Clusters[0].Name, "cluster-a")
	}

	// Should only have user-a
	if len(iso.Users) != 1 {
		t.Fatalf("len(Users) = %d, want 1", len(iso.Users))
	}
	if iso.Users[0].Name != "user-a" {
		t.Errorf("User name = %q, want %q", iso.Users[0].Name, "user-a")
	}
}

func TestBuildIsolatedConfig_MissingCluster(t *testing.T) {
	full := &kubeconfig.KubeConfig{
		Clusters: []kubeconfig.NamedCluster{
			{Name: "other-cluster", Cluster: kubeconfig.Cluster{Server: "https://other:6443"}},
		},
		Contexts: []kubeconfig.NamedContext{
			{Name: "ctx", Context: kubeconfig.Context{Cluster: "missing-cluster", User: "user"}},
		},
		Users: []kubeconfig.NamedUser{
			{Name: "user", User: kubeconfig.User{Token: "tok"}},
		},
	}

	namedCtx := &full.Contexts[0]
	iso := buildIsolatedConfig(full, namedCtx)

	// Cluster should be empty since referenced cluster doesn't exist
	if len(iso.Clusters) != 0 {
		t.Errorf("len(Clusters) = %d, want 0 for missing cluster ref", len(iso.Clusters))
	}
}

func TestBuildEnv(t *testing.T) {
	// Clear yaks vars first
	t.Setenv("KUBECONFIG", "")
	t.Setenv("YAKS_CONTEXT", "")
	t.Setenv("YAKS_NAMESPACE", "")
	t.Setenv("YAKS_DEPTH", "")
	t.Setenv("YAKS_ACTIVE", "")
	t.Setenv("YAKS_KUBECONFIG", "")

	env := buildEnv("/tmp/kc", "my-ctx", "my-ns", 2)

	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	checks := map[string]string{
		"KUBECONFIG":     "/tmp/kc",
		"YAKS_CONTEXT":   "my-ctx",
		"YAKS_NAMESPACE": "my-ns",
		"YAKS_DEPTH":     "2",
		"YAKS_ACTIVE":    "1",
	}

	for key, want := range checks {
		if got, ok := envMap[key]; !ok {
			t.Errorf("missing env var %s", key)
		} else if got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	// YAKS_KUBECONFIG should be set (to default path since KUBECONFIG was empty)
	if _, ok := envMap["YAKS_KUBECONFIG"]; !ok {
		t.Error("missing env var YAKS_KUBECONFIG")
	}
}

func TestBuildEnv_FiltersExisting(t *testing.T) {
	// Set some yaks vars that should be overwritten
	t.Setenv("KUBECONFIG", "/old/config")
	t.Setenv("YAKS_CONTEXT", "old-context")
	t.Setenv("YAKS_NAMESPACE", "old-ns")
	t.Setenv("YAKS_DEPTH", "99")
	t.Setenv("YAKS_ACTIVE", "0")
	t.Setenv("YAKS_KUBECONFIG", "/original/config")

	env := buildEnv("/new/config", "new-ctx", "new-ns", 1)

	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["KUBECONFIG"] != "/new/config" {
		t.Errorf("KUBECONFIG = %q, want /new/config", envMap["KUBECONFIG"])
	}
	if envMap["YAKS_CONTEXT"] != "new-ctx" {
		t.Errorf("YAKS_CONTEXT = %q, want new-ctx", envMap["YAKS_CONTEXT"])
	}
	if envMap["YAKS_DEPTH"] != "1" {
		t.Errorf("YAKS_DEPTH = %q, want 1", envMap["YAKS_DEPTH"])
	}
	// YAKS_KUBECONFIG should carry forward the original, not get replaced
	if envMap["YAKS_KUBECONFIG"] != "/original/config" {
		t.Errorf("YAKS_KUBECONFIG = %q, want /original/config", envMap["YAKS_KUBECONFIG"])
	}

	// Ensure no duplicate keys
	counts := make(map[string]int)
	for _, e := range env {
		key := strings.SplitN(e, "=", 2)[0]
		counts[key]++
	}
	for key, count := range counts {
		if count > 1 {
			t.Errorf("duplicate env var %s (count: %d)", key, count)
		}
	}
}

func TestBuildEnv_PreservesOtherVars(t *testing.T) {
	t.Setenv("MY_CUSTOM_VAR", "hello")
	t.Setenv("PATH", "/usr/bin")

	env := buildEnv("/tmp/kc", "ctx", "ns", 1)

	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["MY_CUSTOM_VAR"] != "hello" {
		t.Errorf("MY_CUSTOM_VAR = %q, want %q", envMap["MY_CUSTOM_VAR"], "hello")
	}
	if envMap["PATH"] == "" {
		t.Error("PATH should be preserved")
	}
}

func TestDetectShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping unix shell detection test on windows")
	}

	// Test with SHELL env var set
	t.Setenv("SHELL", "/bin/zsh")
	got := detectShell()
	if got != "/bin/zsh" {
		t.Errorf("detectShell() = %q, want /bin/zsh", got)
	}

	// Test with SHELL env var set to bash
	t.Setenv("SHELL", "/bin/bash")
	got = detectShell()
	if got != "/bin/bash" {
		t.Errorf("detectShell() = %q, want /bin/bash", got)
	}
}

func TestDetectShell_Fallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping unix shell detection test on windows")
	}

	t.Setenv("SHELL", "")
	got := detectShell()
	// Should return some valid shell path
	if got == "" {
		t.Error("detectShell() returned empty string")
	}
}

func TestSetupIsolatedEnv(t *testing.T) {
	kcPath := writeSampleKubeconfig(t)
	t.Setenv("KUBECONFIG", kcPath)
	t.Setenv("YAKS_DEPTH", "0")

	tmpDir, tmpKubeconfig, ctx, env, err := setupIsolatedEnv("test-ctx", "custom-ns")
	if err != nil {
		t.Fatalf("setupIsolatedEnv() error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Verify tmpDir exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("tmpDir does not exist")
	}

	// Verify temp kubeconfig was written
	if _, err := os.Stat(tmpKubeconfig); os.IsNotExist(err) {
		t.Error("tmpKubeconfig does not exist")
	}

	// Verify context was resolved
	if ctx.Name != "test-ctx" {
		t.Errorf("ctx.Name = %q, want %q", ctx.Name, "test-ctx")
	}
	if ctx.Context.Namespace != "custom-ns" {
		t.Errorf("ctx.Context.Namespace = %q, want %q", ctx.Context.Namespace, "custom-ns")
	}

	// Verify env contains yaks vars
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if envMap["YAKS_CONTEXT"] != "test-ctx" {
		t.Errorf("YAKS_CONTEXT = %q, want test-ctx", envMap["YAKS_CONTEXT"])
	}
	if envMap["YAKS_NAMESPACE"] != "custom-ns" {
		t.Errorf("YAKS_NAMESPACE = %q, want custom-ns", envMap["YAKS_NAMESPACE"])
	}

	// Verify the temp kubeconfig is valid and contains only the isolated context
	loaded, err := kubeconfig.Load(tmpKubeconfig)
	if err != nil {
		t.Fatalf("failed to load temp kubeconfig: %v", err)
	}
	if loaded.CurrentContext != "test-ctx" {
		t.Errorf("temp kubeconfig CurrentContext = %q, want test-ctx", loaded.CurrentContext)
	}
	if len(loaded.Contexts) != 1 {
		t.Errorf("temp kubeconfig has %d contexts, want 1", len(loaded.Contexts))
	}
}

func TestSetupIsolatedEnv_DefaultNamespace(t *testing.T) {
	kcPath := writeSampleKubeconfig(t)
	t.Setenv("KUBECONFIG", kcPath)
	t.Setenv("YAKS_DEPTH", "0")

	tmpDir, _, ctx, _, err := setupIsolatedEnv("test-ctx", "")
	if err != nil {
		t.Fatalf("setupIsolatedEnv() error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// When namespace is empty and context has "default", should keep "default"
	if ctx.Context.Namespace != "default" {
		t.Errorf("namespace = %q, want %q", ctx.Context.Namespace, "default")
	}
}

func TestSetupIsolatedEnv_InvalidContext(t *testing.T) {
	kcPath := writeSampleKubeconfig(t)
	t.Setenv("KUBECONFIG", kcPath)

	_, _, _, _, err := setupIsolatedEnv("nonexistent-ctx", "default")
	if err == nil {
		t.Fatal("setupIsolatedEnv() expected error for nonexistent context, got nil")
	}
}

func TestSetupIsolatedEnv_NestedShell(t *testing.T) {
	// Write a kubeconfig with two contexts
	dir := t.TempDir()
	fullPath := filepath.Join(dir, "config")
	content := `apiVersion: v1
kind: Config
current-context: ctx-a
clusters:
- name: cluster-a
  cluster:
    server: https://a:6443
- name: cluster-b
  cluster:
    server: https://b:6443
contexts:
- name: ctx-a
  context:
    cluster: cluster-a
    user: user-a
    namespace: default
- name: ctx-b
  context:
    cluster: cluster-b
    user: user-b
    namespace: default
users:
- name: user-a
  user:
    token: token-a
- name: user-b
  user:
    token: token-b
`
	if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	// Simulate being inside a yaks shell: KUBECONFIG points to a temp file
	// with only ctx-a, but YAKS_KUBECONFIG has the original full config.
	tmpDir1 := t.TempDir()
	tmpKC := filepath.Join(tmpDir1, "config")
	singleCtx := `apiVersion: v1
kind: Config
current-context: ctx-a
clusters:
- name: cluster-a
  cluster:
    server: https://a:6443
contexts:
- name: ctx-a
  context:
    cluster: cluster-a
    user: user-a
    namespace: default
users:
- name: user-a
  user:
    token: token-a
`
	if err := os.WriteFile(tmpKC, []byte(singleCtx), 0600); err != nil {
		t.Fatalf("failed to write temp kubeconfig: %v", err)
	}

	// KUBECONFIG = temp (only ctx-a), YAKS_KUBECONFIG = full (ctx-a + ctx-b)
	t.Setenv("KUBECONFIG", tmpKC)
	t.Setenv("YAKS_KUBECONFIG", fullPath)
	t.Setenv("YAKS_DEPTH", "1")
	t.Setenv("YAKS_ACTIVE", "1")

	// Should be able to switch to ctx-b even though KUBECONFIG only has ctx-a
	tmpDir2, _, ctx, _, err := setupIsolatedEnv("ctx-b", "kube-system")
	if err != nil {
		t.Fatalf("nested setupIsolatedEnv(ctx-b) error: %v — nesting is broken", err)
	}
	defer os.RemoveAll(tmpDir2)

	if ctx.Name != "ctx-b" {
		t.Errorf("ctx.Name = %q, want ctx-b", ctx.Name)
	}
	if ctx.Context.Namespace != "kube-system" {
		t.Errorf("namespace = %q, want kube-system", ctx.Context.Namespace)
	}
}

func TestExecCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}

	kcPath := writeSampleKubeconfig(t)
	t.Setenv("KUBECONFIG", kcPath)
	t.Setenv("YAKS_DEPTH", "0")

	// Run a simple echo command
	exitCode, err := ExecCommand("test-ctx", "default", []string{"echo", "hello"})
	if err != nil {
		t.Fatalf("ExecCommand() error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
}

func TestExecCommand_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}

	kcPath := writeSampleKubeconfig(t)
	t.Setenv("KUBECONFIG", kcPath)
	t.Setenv("YAKS_DEPTH", "0")

	// Run a command that exits with non-zero
	exitCode, err := ExecCommand("test-ctx", "default", []string{"sh", "-c", "exit 42"})
	if err != nil {
		t.Fatalf("ExecCommand() error: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("exitCode = %d, want 42", exitCode)
	}
}

func TestExecCommand_InvalidContext(t *testing.T) {
	kcPath := writeSampleKubeconfig(t)
	t.Setenv("KUBECONFIG", kcPath)

	_, err := ExecCommand("nonexistent", "default", []string{"echo", "test"})
	if err == nil {
		t.Fatal("ExecCommand() expected error for nonexistent context, got nil")
	}
}

func TestExecCommand_EnvIsolation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping exec test on windows")
	}

	kcPath := writeSampleKubeconfig(t)
	t.Setenv("KUBECONFIG", kcPath)
	t.Setenv("YAKS_DEPTH", "0")

	// Verify the spawned env has the correct YAKS vars
	// by running printenv and checking output
	exitCode, err := ExecCommand("test-ctx", "my-namespace", []string{
		"sh", "-c", fmt.Sprintf(`
			test "$YAKS_ACTIVE" = "1" &&
			test "$YAKS_CONTEXT" = "test-ctx" &&
			test "$YAKS_NAMESPACE" = "my-namespace" &&
			test "$KUBECONFIG" != "%s"
		`, kcPath),
	})
	if err != nil {
		t.Fatalf("ExecCommand() error: %v", err)
	}
	if exitCode != 0 {
		t.Error("environment isolation checks failed — expected YAKS vars to be set correctly")
	}
}

func TestEnvScript_Fish(t *testing.T) {
	got := EnvScript("fish", "/tmp/yaks-123", "/tmp/yaks-123/config", "/home/user/.kube/config", "prod", "default")
	if !strings.Contains(got, "set -gx KUBECONFIG") {
		t.Error("fish EnvScript missing KUBECONFIG")
	}
	if !strings.Contains(got, "set -gx YAKS_CONTEXT") {
		t.Error("fish EnvScript missing YAKS_CONTEXT")
	}
	if !strings.Contains(got, "set -gx YAKS_NAMESPACE") {
		t.Error("fish EnvScript missing YAKS_NAMESPACE")
	}
	if !strings.Contains(got, "set -gx YAKS_ACTIVE 1") {
		t.Error("fish EnvScript missing YAKS_ACTIVE")
	}
	if !strings.Contains(got, "set -gx YAKS_TMPDIR") {
		t.Error("fish EnvScript missing YAKS_TMPDIR")
	}
	if !strings.Contains(got, "set -gx YAKS_KUBECONFIG") {
		t.Error("fish EnvScript missing YAKS_KUBECONFIG")
	}
}

func TestEnvScript_Bash(t *testing.T) {
	got := EnvScript("bash", "/tmp/yaks-123", "/tmp/yaks-123/config", "/home/user/.kube/config", "prod", "kube-system")
	if !strings.Contains(got, "export KUBECONFIG=") {
		t.Error("bash EnvScript missing KUBECONFIG")
	}
	if !strings.Contains(got, "export YAKS_CONTEXT=") {
		t.Error("bash EnvScript missing YAKS_CONTEXT")
	}
	if !strings.Contains(got, "export YAKS_ACTIVE=1") {
		t.Error("bash EnvScript missing YAKS_ACTIVE")
	}
	if !strings.Contains(got, "export YAKS_TMPDIR=") {
		t.Error("bash EnvScript missing YAKS_TMPDIR")
	}
}

func TestEnvScript_Zsh(t *testing.T) {
	got := EnvScript("zsh", "/tmp/yaks-123", "/tmp/yaks-123/config", "/home/user/.kube/config", "staging", "monitoring")
	if !strings.Contains(got, "export KUBECONFIG=") {
		t.Error("zsh EnvScript missing KUBECONFIG")
	}
	if !strings.Contains(got, "'staging'") {
		t.Error("zsh EnvScript missing quoted context name")
	}
}

func TestEnvScript_PowerShell(t *testing.T) {
	got := EnvScript("powershell", "/tmp/yaks-123", "/tmp/yaks-123/config", "/home/.kube/config", "ctx", "ns")
	if !strings.Contains(got, "$env:KUBECONFIG") {
		t.Error("powershell EnvScript missing KUBECONFIG")
	}
	if !strings.Contains(got, "$env:YAKS_CONTEXT") {
		t.Error("powershell EnvScript missing YAKS_CONTEXT")
	}
	if !strings.Contains(got, "$env:YAKS_ACTIVE = '1'") {
		t.Error("powershell EnvScript missing YAKS_ACTIVE")
	}
	if !strings.Contains(got, "$env:YAKS_TMPDIR") {
		t.Error("powershell EnvScript missing YAKS_TMPDIR")
	}
}

func TestEnvScript_Unsupported(t *testing.T) {
	got := EnvScript("tcsh", "/tmp", "/tmp/config", "/home/.kube/config", "ctx", "ns")
	if got != "" {
		t.Errorf("EnvScript(tcsh) = %q, want empty", got)
	}
}

func TestNsEnvScript_Fish(t *testing.T) {
	got := NsEnvScript("fish", "kube-system")
	if !strings.Contains(got, "set -gx YAKS_NAMESPACE") {
		t.Error("fish NsEnvScript missing YAKS_NAMESPACE")
	}
	if !strings.Contains(got, "'kube-system'") {
		t.Error("fish NsEnvScript missing quoted namespace")
	}
}

func TestNsEnvScript_Bash(t *testing.T) {
	got := NsEnvScript("bash", "monitoring")
	if !strings.Contains(got, "export YAKS_NAMESPACE=") {
		t.Error("bash NsEnvScript missing YAKS_NAMESPACE")
	}
}

func TestNsEnvScript_PowerShell(t *testing.T) {
	got := NsEnvScript("powershell", "kube-system")
	if !strings.Contains(got, "$env:YAKS_NAMESPACE") {
		t.Error("powershell NsEnvScript missing YAKS_NAMESPACE")
	}
}

func TestPsQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with spaces", "'with spaces'"},
		{"it's", "'it''s'"},
		{"", "''"},
	}
	for _, tt := range tests {
		got := psQuote(tt.input)
		if got != tt.want {
			t.Errorf("psQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with spaces", "'with spaces'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
	}
	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestOriginalKubeconfig_Default(t *testing.T) {
	t.Setenv("YAKS_KUBECONFIG", "")
	t.Setenv("KUBECONFIG", "")
	got := OriginalKubeconfig()
	if got == "" {
		t.Error("OriginalKubeconfig() returned empty, want default path")
	}
}

func TestOriginalKubeconfig_YaksKubeconfig(t *testing.T) {
	t.Setenv("YAKS_KUBECONFIG", "/original/config")
	t.Setenv("KUBECONFIG", "/isolated/config")
	got := OriginalKubeconfig()
	if got != "/original/config" {
		t.Errorf("OriginalKubeconfig() = %q, want %q", got, "/original/config")
	}
}

func TestOriginalKubeconfig_Kubeconfig(t *testing.T) {
	t.Setenv("YAKS_KUBECONFIG", "")
	t.Setenv("KUBECONFIG", "/my/kubeconfig")
	got := OriginalKubeconfig()
	if got != "/my/kubeconfig" {
		t.Errorf("OriginalKubeconfig() = %q, want %q", got, "/my/kubeconfig")
	}
}
