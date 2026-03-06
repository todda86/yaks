package fzf

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsAvailable checks if fzf is installed on the system.
func IsAvailable() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// Select presents a list of items via fzf and returns the selected item.
// If fzf is not available, falls back to a simple numbered list.
func Select(items []string, prompt string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select from")
	}

	if IsAvailable() {
		return selectWithFzf(items, prompt)
	}

	return selectWithList(items, prompt)
}

func selectWithFzf(items []string, prompt string) (string, error) {
	cmd := exec.Command("fzf", "--prompt", prompt+" > ", "--height", "40%", "--reverse")
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", fmt.Errorf("selection cancelled")
		}
		return "", fmt.Errorf("fzf error: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func selectWithList(items []string, prompt string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s:\n", prompt)
	for i, item := range items {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, item)
	}

	fmt.Fprint(os.Stderr, "\nEnter number: ")
	var choice int
	_, err := fmt.Scanf("%d", &choice)
	if err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if choice < 1 || choice > len(items) {
		return "", fmt.Errorf("choice %d out of range (1-%d)", choice, len(items))
	}

	return items[choice-1], nil
}
