package fzf

import (
	"testing"
)

func TestIsAvailable(t *testing.T) {
	_ = IsAvailable()
}

func TestSelect_EmptyList(t *testing.T) {
	_, err := Select([]string{}, "Pick one")
	if err == nil {
		t.Fatal("Select() expected error for empty list, got nil")
	}
}
