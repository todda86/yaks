package kubeconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeTempKubeconfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp kubeconfig: %v", err)
	}
	return path
}

const sampleKubeconfig = `apiVersion: v1
kind: Config
current-context: dev
clusters:
- name: dev-cluster
  cluster:
    server: https://dev.example.com:6443
- name: prod-cluster
  cluster:
    server: https://prod.example.com:6443
    insecure-skip-tls-verify: true
contexts:
- name: dev
  context:
    cluster: dev-cluster
    user: dev-user
    namespace: default
- name: prod
  context:
    cluster: prod-cluster
    user: prod-user
    namespace: kube-system
users:
- name: dev-user
  user:
    token: dev-token-123
- name: prod-user
  user:
    token: prod-token-456
`

const overlayKubeconfig = `apiVersion: v1
kind: Config
current-context: staging
clusters:
- name: staging-cluster
  cluster:
    server: https://staging.example.com:6443
- name: dev-cluster
  cluster:
    server: https://should-be-ignored.example.com
contexts:
- name: staging
  context:
    cluster: staging-cluster
    user: staging-user
    namespace: staging-ns
- name: dev
  context:
    cluster: dev-cluster
    user: dev-user
users:
- name: staging-user
  user:
    token: staging-token
- name: dev-user
  user:
    token: should-be-ignored
`

func TestLoad(t *testing.T) {
	path := writeTempKubeconfig(t, sampleKubeconfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.APIVersion != "v1" {
		t.Errorf("APIVersion = %q, want %q", cfg.APIVersion, "v1")
	}
	if cfg.Kind != "Config" {
		t.Errorf("Kind = %q, want %q", cfg.Kind, "Config")
	}
	if cfg.CurrentContext != "dev" {
		t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, "dev")
	}
	if len(cfg.Clusters) != 2 {
		t.Errorf("len(Clusters) = %d, want 2", len(cfg.Clusters))
	}
	if len(cfg.Contexts) != 2 {
		t.Errorf("len(Contexts) = %d, want 2", len(cfg.Contexts))
	}
	if len(cfg.Users) != 2 {
		t.Errorf("len(Users) = %d, want 2", len(cfg.Users))
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config")
	if err == nil {
		t.Fatal("Load() expected error for nonexistent file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTempKubeconfig(t, "not: [valid: yaml: {{")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	}
}

func TestSave(t *testing.T) {
	cfg := &KubeConfig{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: "test",
		Clusters: []NamedCluster{
			{Name: "test-cluster", Cluster: Cluster{Server: "https://localhost:6443"}},
		},
		Contexts: []NamedContext{
			{Name: "test", Context: Context{Cluster: "test-cluster", User: "test-user", Namespace: "default"}},
		},
		Users: []NamedUser{
			{Name: "test-user", User: User{Token: "abc123"}},
		},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config")
	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}
	if loaded.CurrentContext != "test" {
		t.Errorf("CurrentContext = %q, want %q", loaded.CurrentContext, "test")
	}
	if len(loaded.Clusters) != 1 {
		t.Errorf("len(Clusters) = %d, want 1", len(loaded.Clusters))
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat() error: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("file permissions = %o, want 0600", perm)
		}
	}
}

func TestListContextNames(t *testing.T) {
	path := writeTempKubeconfig(t, sampleKubeconfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	names := cfg.ListContextNames()
	if len(names) != 2 {
		t.Fatalf("len(names) = %d, want 2", len(names))
	}
	expected := map[string]bool{"dev": true, "prod": true}
	for _, n := range names {
		if !expected[n] {
			t.Errorf("unexpected context name: %q", n)
		}
	}
}

func TestListContextNames_Empty(t *testing.T) {
	cfg := &KubeConfig{}
	names := cfg.ListContextNames()
	if len(names) != 0 {
		t.Errorf("len(names) = %d, want 0", len(names))
	}
}

func TestGetContext(t *testing.T) {
	path := writeTempKubeconfig(t, sampleKubeconfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	ctx, err := cfg.GetContext("prod")
	if err != nil {
		t.Fatalf("GetContext() error: %v", err)
	}
	if ctx.Name != "prod" {
		t.Errorf("Name = %q, want %q", ctx.Name, "prod")
	}
	if ctx.Context.Cluster != "prod-cluster" {
		t.Errorf("Cluster = %q, want %q", ctx.Context.Cluster, "prod-cluster")
	}
	if ctx.Context.User != "prod-user" {
		t.Errorf("User = %q, want %q", ctx.Context.User, "prod-user")
	}
	if ctx.Context.Namespace != "kube-system" {
		t.Errorf("Namespace = %q, want %q", ctx.Context.Namespace, "kube-system")
	}
}

func TestGetContext_NotFound(t *testing.T) {
	path := writeTempKubeconfig(t, sampleKubeconfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	_, err = cfg.GetContext("nonexistent")
	if err == nil {
		t.Fatal("GetContext() expected error for nonexistent context, got nil")
	}
}

func TestGetCluster(t *testing.T) {
	path := writeTempKubeconfig(t, sampleKubeconfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	cluster, err := cfg.GetCluster("prod-cluster")
	if err != nil {
		t.Fatalf("GetCluster() error: %v", err)
	}
	if cluster.Name != "prod-cluster" {
		t.Errorf("Name = %q, want %q", cluster.Name, "prod-cluster")
	}
	if cluster.Cluster.Server != "https://prod.example.com:6443" {
		t.Errorf("Server = %q, want %q", cluster.Cluster.Server, "https://prod.example.com:6443")
	}
	if !cluster.Cluster.InsecureSkipTLSVerify {
		t.Error("InsecureSkipTLSVerify = false, want true")
	}
}

func TestGetCluster_NotFound(t *testing.T) {
	path := writeTempKubeconfig(t, sampleKubeconfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	_, err = cfg.GetCluster("nonexistent")
	if err == nil {
		t.Fatal("GetCluster() expected error for nonexistent cluster, got nil")
	}
}

func TestSetNamespace(t *testing.T) {
	path := writeTempKubeconfig(t, sampleKubeconfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if err := cfg.SetNamespace("dev", "monitoring"); err != nil {
		t.Fatalf("SetNamespace() error: %v", err)
	}
	ctx, _ := cfg.GetContext("dev")
	if ctx.Context.Namespace != "monitoring" {
		t.Errorf("Namespace = %q, want %q", ctx.Context.Namespace, "monitoring")
	}
}

func TestSetNamespace_NotFound(t *testing.T) {
	path := writeTempKubeconfig(t, sampleKubeconfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	err = cfg.SetNamespace("nonexistent", "default")
	if err == nil {
		t.Fatal("SetNamespace() expected error for nonexistent context, got nil")
	}
}

func TestMerge(t *testing.T) {
	basePath := writeTempKubeconfig(t, sampleKubeconfig)
	overlayPath := writeTempKubeconfig(t, overlayKubeconfig)
	base, err := Load(basePath)
	if err != nil {
		t.Fatalf("Load(base) error: %v", err)
	}
	overlay, err := Load(overlayPath)
	if err != nil {
		t.Fatalf("Load(overlay) error: %v", err)
	}
	merged := merge(base, overlay)
	if len(merged.Clusters) != 3 {
		t.Errorf("len(Clusters) = %d, want 3", len(merged.Clusters))
	}
	if len(merged.Contexts) != 3 {
		t.Errorf("len(Contexts) = %d, want 3", len(merged.Contexts))
	}
	if len(merged.Users) != 3 {
		t.Errorf("len(Users) = %d, want 3", len(merged.Users))
	}
	devCluster, err := merged.GetCluster("dev-cluster")
	if err != nil {
		t.Fatalf("GetCluster(dev-cluster) error: %v", err)
	}
	if devCluster.Cluster.Server != "https://dev.example.com:6443" {
		t.Errorf("dev-cluster Server = %q, want base value", devCluster.Cluster.Server)
	}
	if merged.CurrentContext != "dev" {
		t.Errorf("CurrentContext = %q, want %q", merged.CurrentContext, "dev")
	}
}

func TestLoadAll(t *testing.T) {
	basePath := writeTempKubeconfig(t, sampleKubeconfig)
	overlayPath := writeTempKubeconfig(t, overlayKubeconfig)
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	t.Setenv("KUBECONFIG", basePath+sep+overlayPath)
	cfg, primaryPath, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}
	if primaryPath != basePath {
		t.Errorf("primaryPath = %q, want %q", primaryPath, basePath)
	}
	names := cfg.ListContextNames()
	if len(names) != 3 {
		t.Errorf("len(contexts) = %d, want 3", len(names))
	}
}

func TestDefaultKubeconfigPath_EnvVar(t *testing.T) {
	t.Setenv("YAKS_KUBECONFIG", "")
	t.Setenv("KUBECONFIG", "/custom/path/config")
	path := DefaultKubeconfigPath()
	if path != "/custom/path/config" {
		t.Errorf("DefaultKubeconfigPath() = %q, want %q", path, "/custom/path/config")
	}
}

func TestDefaultKubeconfigPath_MultipleEnvVar(t *testing.T) {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	t.Setenv("YAKS_KUBECONFIG", "")
	t.Setenv("KUBECONFIG", "/first/config"+sep+"/second/config")
	path := DefaultKubeconfigPath()
	if path != "/first/config" {
		t.Errorf("DefaultKubeconfigPath() = %q, want %q", path, "/first/config")
	}
}

func TestDefaultKubeconfigPath_NoEnvVar(t *testing.T) {
	t.Setenv("YAKS_KUBECONFIG", "")
	t.Setenv("KUBECONFIG", "")
	path := DefaultKubeconfigPath()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".kube", "config")
	if path != expected {
		t.Errorf("DefaultKubeconfigPath() = %q, want %q", path, expected)
	}
}

func TestDefaultKubeconfigPath_YaksKubeconfigPreferred(t *testing.T) {
	t.Setenv("KUBECONFIG", "/tmp/yaks-12345/config")
	t.Setenv("YAKS_KUBECONFIG", "/original/config")
	path := DefaultKubeconfigPath()
	if path != "/original/config" {
		t.Errorf("DefaultKubeconfigPath() = %q, want /original/config", path)
	}
}

func TestAllKubeconfigPaths(t *testing.T) {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	t.Setenv("YAKS_KUBECONFIG", "")
	t.Setenv("KUBECONFIG", "/a/config"+sep+"/b/config"+sep+"  "+sep+"/c/config")
	paths := AllKubeconfigPaths()
	if len(paths) != 3 {
		t.Fatalf("len(paths) = %d, want 3", len(paths))
	}
	if paths[0] != "/a/config" || paths[1] != "/b/config" || paths[2] != "/c/config" {
		t.Errorf("paths = %v, want [/a/config /b/config /c/config]", paths)
	}
}

func TestAllKubeconfigPaths_NoEnvVar(t *testing.T) {
	t.Setenv("YAKS_KUBECONFIG", "")
	t.Setenv("KUBECONFIG", "")
	paths := AllKubeconfigPaths()
	if len(paths) != 1 {
		t.Fatalf("len(paths) = %d, want 1", len(paths))
	}
}

func TestAllKubeconfigPaths_YaksKubeconfigPreferred(t *testing.T) {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	t.Setenv("KUBECONFIG", "/tmp/yaks-12345/config")
	t.Setenv("YAKS_KUBECONFIG", "/original/a"+sep+"/original/b")
	paths := AllKubeconfigPaths()
	if len(paths) != 2 {
		t.Fatalf("len(paths) = %d, want 2", len(paths))
	}
	if paths[0] != "/original/a" || paths[1] != "/original/b" {
		t.Errorf("paths = %v, want [/original/a /original/b]", paths)
	}
}

func TestSaveAndReload_RoundTrip(t *testing.T) {
	cfg := &KubeConfig{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: "round-trip",
		Clusters: []NamedCluster{
			{
				Name: "rt-cluster",
				Cluster: Cluster{
					Server:                "https://localhost:6443",
					InsecureSkipTLSVerify: true,
				},
			},
		},
		Contexts: []NamedContext{
			{
				Name: "round-trip",
				Context: Context{
					Cluster:   "rt-cluster",
					User:      "rt-user",
					Namespace: "rt-ns",
				},
			},
		},
		Users: []NamedUser{
			{
				Name: "rt-user",
				User: User{
					Token: "rt-token",
					Exec: &ExecConfig{
						APIVersion: "client.authentication.k8s.io/v1beta1",
						Command:    "aws",
						Args:       []string{"eks", "get-token", "--cluster-name", "my-cluster"},
						Env: []ExecEnvVar{
							{Name: "AWS_PROFILE", Value: "prod"},
						},
					},
				},
			},
		},
	}
	path := filepath.Join(t.TempDir(), "config")
	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.CurrentContext != cfg.CurrentContext {
		t.Errorf("CurrentContext = %q, want %q", loaded.CurrentContext, cfg.CurrentContext)
	}
	if loaded.Users[0].User.Exec == nil {
		t.Fatal("User Exec config is nil after round-trip")
	}
	if loaded.Users[0].User.Exec.Command != "aws" {
		t.Errorf("Exec.Command = %q, want %q", loaded.Users[0].User.Exec.Command, "aws")
	}
	if len(loaded.Users[0].User.Exec.Args) != 4 {
		t.Errorf("len(Exec.Args) = %d, want 4", len(loaded.Users[0].User.Exec.Args))
	}
	if len(loaded.Users[0].User.Exec.Env) != 1 || loaded.Users[0].User.Exec.Env[0].Value != "prod" {
		t.Error("Exec.Env not preserved in round-trip")
	}
	if !loaded.Clusters[0].Cluster.InsecureSkipTLSVerify {
		t.Error("InsecureSkipTLSVerify not preserved in round-trip")
	}
}
