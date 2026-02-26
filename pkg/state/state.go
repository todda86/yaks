package state

import (
	"os"
	"strconv"
)

// IsActive returns true if we're inside a yaks-managed shell.
func IsActive() bool {
	return os.Getenv("YAKS_ACTIVE") == "1"
}

// GetCurrentContext returns the current yaks context name, if inside a managed shell.
func GetCurrentContext() string {
	return os.Getenv("YAKS_CONTEXT")
}

// GetCurrentNamespace returns the current yaks namespace, if inside a managed shell.
func GetCurrentNamespace() string {
	return os.Getenv("YAKS_NAMESPACE")
}

// Quiet returns true if status messages should be suppressed.
// Set YAKS_SILENT=1 to suppress.
func Quiet() bool {
	return os.Getenv("YAKS_SILENT") == "1"
}

// NoPrompt returns true if the shell prompt segment should be suppressed.
// Set YAKS_NO_PROMPT=1 to suppress.
func NoPrompt() bool {
	return os.Getenv("YAKS_NO_PROMPT") == "1"
}

// GetDepth returns the current nesting depth of yaks shells.
func GetDepth() int {
	d, err := strconv.Atoi(os.Getenv("YAKS_DEPTH"))
	if err != nil {
		return 0
	}
	return d
}
