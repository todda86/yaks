package state

import (
	"testing"
)

func TestIsActive(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"active", "1", true},
		{"inactive empty", "", false},
		{"inactive zero", "0", false},
		{"inactive other", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("YAKS_ACTIVE", tt.value)
			if got := IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCurrentContext(t *testing.T) {
	t.Setenv("YAKS_CONTEXT", "my-cluster")
	if got := GetCurrentContext(); got != "my-cluster" {
		t.Errorf("GetCurrentContext() = %q, want %q", got, "my-cluster")
	}

	t.Setenv("YAKS_CONTEXT", "")
	if got := GetCurrentContext(); got != "" {
		t.Errorf("GetCurrentContext() = %q, want empty", got)
	}
}

func TestGetCurrentNamespace(t *testing.T) {
	t.Setenv("YAKS_NAMESPACE", "kube-system")
	if got := GetCurrentNamespace(); got != "kube-system" {
		t.Errorf("GetCurrentNamespace() = %q, want %q", got, "kube-system")
	}

	t.Setenv("YAKS_NAMESPACE", "")
	if got := GetCurrentNamespace(); got != "" {
		t.Errorf("GetCurrentNamespace() = %q, want empty", got)
	}
}

func TestQuiet(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"silent", "1", true},
		{"not silent empty", "", false},
		{"not silent zero", "0", false},
		{"not silent other", "true", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("YAKS_SILENT", tt.value)
			if got := Quiet(); got != tt.want {
				t.Errorf("Quiet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDepth(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"zero", "0", 0},
		{"one", "1", 1},
		{"five", "5", 5},
		{"empty", "", 0},
		{"invalid", "abc", 0},
		{"negative", "-1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("YAKS_DEPTH", tt.value)
			if got := GetDepth(); got != tt.want {
				t.Errorf("GetDepth() = %d, want %d", got, tt.want)
			}
		})
	}
}
