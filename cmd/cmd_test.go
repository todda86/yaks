package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root --help error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "yaks") {
		t.Error("help output missing 'yaks'")
	}
	if !strings.Contains(out, "context") {
		t.Error("help output missing context subcommand")
	}
}

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version error: %v", err)
	}
}

func TestInitCommand_Bash(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "bash"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init bash error: %v", err)
	}
}

func TestInitCommand_Zsh(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "zsh"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init zsh error: %v", err)
	}
}

func TestInitCommand_Fish(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "fish"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init fish error: %v", err)
	}
}

func TestInitCommand_Unsupported(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "tcsh"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("init tcsh expected error, got nil")
	}
}

func TestCompletionCommand_Bash(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"completion", "bash"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion bash error: %v", err)
	}
}

func TestCompletionCommand_Zsh(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"completion", "zsh"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion zsh error: %v", err)
	}
}

func TestCompletionCommand_Fish(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"completion", "fish"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion fish error: %v", err)
	}
}

func TestCompletionCommand_Powershell(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"completion", "powershell"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion powershell error: %v", err)
	}
}

func TestSubcommands_Registered(t *testing.T) {
	expected := []string{"ctx", "ns", "exec", "info", "init", "list", "version", "completion"}
	cmdNames := make(map[string]bool)
	for _, sub := range rootCmd.Commands() {
		cmdNames[sub.Name()] = true
	}

	for _, name := range expected {
		if !cmdNames[name] {
			t.Errorf("subcommand %q not registered on root", name)
		}
	}
}

func TestCtxCommand_Aliases(t *testing.T) {
	found := false
	for _, alias := range ctxCmd.Aliases {
		if alias == "context" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ctx command missing 'context' alias")
	}
}

func TestNsCommand_Aliases(t *testing.T) {
	found := false
	for _, alias := range nsCmd.Aliases {
		if alias == "namespace" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ns command missing 'namespace' alias")
	}
}

func TestExecCommand_HasNamespaceFlag(t *testing.T) {
	flag := execCmd.Flags().Lookup("namespace")
	if flag == nil {
		t.Fatal("exec command missing --namespace flag")
	}
	if flag.Shorthand != "n" {
		t.Errorf("namespace flag shorthand = %q, want %q", flag.Shorthand, "n")
	}
}

func TestCtxCommand_HasNamespaceFlag(t *testing.T) {
	flag := ctxCmd.Flags().Lookup("namespace")
	if flag == nil {
		t.Fatal("ctx command missing --namespace flag")
	}
	if flag.Shorthand != "n" {
		t.Errorf("namespace flag shorthand = %q, want %q", flag.Shorthand, "n")
	}
}

func TestCtxCommand_HasShellEvalFlag(t *testing.T) {
	flag := ctxCmd.Flags().Lookup("shell-eval")
	if flag == nil {
		t.Fatal("ctx command missing --shell-eval flag")
	}
	if flag.DefValue != "" {
		t.Errorf("shell-eval flag default = %q, want empty", flag.DefValue)
	}
}

func TestNsCommand_HasShellEvalFlag(t *testing.T) {
	flag := nsCmd.Flags().Lookup("shell-eval")
	if flag == nil {
		t.Fatal("ns command missing --shell-eval flag")
	}
	if flag.DefValue != "" {
		t.Errorf("shell-eval flag default = %q, want empty", flag.DefValue)
	}
}

func TestListCommand_HasSubcommands(t *testing.T) {
	subNames := make(map[string]bool)
	for _, sub := range listCmd.Commands() {
		subNames[sub.Name()] = true
	}

	if !subNames["contexts"] {
		t.Error("list command missing 'contexts' subcommand")
	}
	if !subNames["namespaces"] {
		t.Error("list command missing 'namespaces' subcommand")
	}
}
