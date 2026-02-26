package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// KubeConfig represents the structure of a kubeconfig file.
type KubeConfig struct {
	APIVersion     string         `yaml:"apiVersion"`
	Kind           string         `yaml:"kind"`
	CurrentContext string         `yaml:"current-context"`
	Clusters       []NamedCluster `yaml:"clusters"`
	Contexts       []NamedContext `yaml:"contexts"`
	Users          []NamedUser    `yaml:"users"`
	Preferences    map[string]any `yaml:"preferences,omitempty"`
}

// NamedCluster associates a name with a cluster config.
type NamedCluster struct {
	Name    string  `yaml:"name"`
	Cluster Cluster `yaml:"cluster"`
}

// Cluster holds the cluster connection information.
type Cluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthority     string `yaml:"certificate-authority,omitempty"`
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify,omitempty"`
}

// NamedContext associates a name with a context config.
type NamedContext struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

// Context holds the context information.
type Context struct {
	Cluster   string `yaml:"cluster"`
	User      string `yaml:"user"`
	Namespace string `yaml:"namespace,omitempty"`
}

// NamedUser associates a name with a user config.
type NamedUser struct {
	Name string `yaml:"name"`
	User User   `yaml:"user"`
}

// User holds user authentication information.
type User struct {
	ClientCertificate     string        `yaml:"client-certificate,omitempty"`
	ClientCertificateData string        `yaml:"client-certificate-data,omitempty"`
	ClientKey             string        `yaml:"client-key,omitempty"`
	ClientKeyData         string        `yaml:"client-key-data,omitempty"`
	Token                 string        `yaml:"token,omitempty"`
	Username              string        `yaml:"username,omitempty"`
	Password              string        `yaml:"password,omitempty"`
	Exec                  *ExecConfig   `yaml:"exec,omitempty"`
	AuthProvider          *AuthProvider `yaml:"auth-provider,omitempty"`
}

// ExecConfig holds exec-based auth plugin configuration.
type ExecConfig struct {
	APIVersion string       `yaml:"apiVersion,omitempty"`
	Command    string       `yaml:"command"`
	Args       []string     `yaml:"args,omitempty"`
	Env        []ExecEnvVar `yaml:"env,omitempty"`
}

// ExecEnvVar holds env vars for exec auth plugins.
type ExecEnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// AuthProvider holds auth provider configuration.
type AuthProvider struct {
	Name   string            `yaml:"name"`
	Config map[string]string `yaml:"config,omitempty"`
}

// DefaultKubeconfigPath returns the default kubeconfig path for the current OS.
func DefaultKubeconfigPath() string {
	// Prefer YAKS_KUBECONFIG when inside a yaks shell so we see all contexts,
	// not just the temporary single-context file that KUBECONFIG points to.
	env := os.Getenv("YAKS_KUBECONFIG")
	if env == "" {
		env = os.Getenv("KUBECONFIG")
	}
	if env != "" {
		// Return the first path if multiple are specified
		sep := ":"
		if runtime.GOOS == "windows" {
			sep = ";"
		}
		paths := strings.Split(env, sep)
		if len(paths) > 0 {
			return paths[0]
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".kube", "config")
	}
	return filepath.Join(home, ".kube", "config")
}

// AllKubeconfigPaths returns all kubeconfig paths from the KUBECONFIG env var
// or the default path if the env var is not set.
// When running inside a yaks shell, YAKS_KUBECONFIG is preferred so that
// nested invocations can see all original contexts instead of only the
// temporary single-context file.
func AllKubeconfigPaths() []string {
	// Prefer YAKS_KUBECONFIG (original paths) over KUBECONFIG (temp file).
	env := os.Getenv("YAKS_KUBECONFIG")
	if env == "" {
		env = os.Getenv("KUBECONFIG")
	}
	if env != "" {
		sep := ":"
		if runtime.GOOS == "windows" {
			sep = ";"
		}
		paths := strings.Split(env, sep)
		var result []string
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	return []string{DefaultKubeconfigPath()}
}

// Load reads and parses a kubeconfig file.
func Load(path string) (*KubeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig %s: %w", path, err)
	}

	var config KubeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig %s: %w", path, err)
	}

	return &config, nil
}

// LoadAll loads and merges all kubeconfig files from the KUBECONFIG paths.
func LoadAll() (*KubeConfig, string, error) {
	paths := AllKubeconfigPaths()
	if len(paths) == 0 {
		return nil, "", fmt.Errorf("no kubeconfig paths found")
	}

	merged, err := Load(paths[0])
	if err != nil {
		return nil, "", err
	}

	for _, path := range paths[1:] {
		cfg, err := Load(path)
		if err != nil {
			// Skip files that can't be loaded when merging
			continue
		}
		merged = merge(merged, cfg)
	}

	return merged, paths[0], nil
}

// Save writes a kubeconfig to a file.
func Save(config *KubeConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig %s: %w", path, err)
	}

	return nil
}

// ListContextNames returns a list of all context names.
func (k *KubeConfig) ListContextNames() []string {
	names := make([]string, 0, len(k.Contexts))
	for _, ctx := range k.Contexts {
		names = append(names, ctx.Name)
	}
	return names
}

// GetContext returns the named context, or an error if not found.
func (k *KubeConfig) GetContext(name string) (*NamedContext, error) {
	for i := range k.Contexts {
		if k.Contexts[i].Name == name {
			return &k.Contexts[i], nil
		}
	}
	return nil, fmt.Errorf("context %q not found", name)
}

// GetCluster returns the named cluster, or an error if not found.
func (k *KubeConfig) GetCluster(name string) (*NamedCluster, error) {
	for i := range k.Clusters {
		if k.Clusters[i].Name == name {
			return &k.Clusters[i], nil
		}
	}
	return nil, fmt.Errorf("cluster %q not found", name)
}

// SetNamespace sets the namespace for a given context.
func (k *KubeConfig) SetNamespace(contextName, namespace string) error {
	for i := range k.Contexts {
		if k.Contexts[i].Name == contextName {
			k.Contexts[i].Context.Namespace = namespace
			return nil
		}
	}
	return fmt.Errorf("context %q not found", contextName)
}

// merge combines two kubeconfigs, with the first taking precedence on conflicts.
func merge(base, overlay *KubeConfig) *KubeConfig {
	// Merge clusters
	existingClusters := make(map[string]bool)
	for _, c := range base.Clusters {
		existingClusters[c.Name] = true
	}
	for _, c := range overlay.Clusters {
		if !existingClusters[c.Name] {
			base.Clusters = append(base.Clusters, c)
		}
	}

	// Merge contexts
	existingContexts := make(map[string]bool)
	for _, c := range base.Contexts {
		existingContexts[c.Name] = true
	}
	for _, c := range overlay.Contexts {
		if !existingContexts[c.Name] {
			base.Contexts = append(base.Contexts, c)
		}
	}

	// Merge users
	existingUsers := make(map[string]bool)
	for _, u := range base.Users {
		existingUsers[u.Name] = true
	}
	for _, u := range overlay.Users {
		if !existingUsers[u.Name] {
			base.Users = append(base.Users, u)
		}
	}

	return base
}
