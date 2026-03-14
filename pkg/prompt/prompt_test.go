package prompt

import (
	"strings"
	"testing"
)

func TestPromptSegment_Inactive(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "")
	if got := PromptSegment(); got != "" {
		t.Errorf("PromptSegment() = %q, want empty when inactive", got)
	}
}

func TestPromptSegment_Active(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "1")
	t.Setenv("YAKS_CONTEXT", "prod")
	t.Setenv("YAKS_NAMESPACE", "monitoring")

	got := PromptSegment()
	want := "[prod|monitoring]"
	if got != want {
		t.Errorf("PromptSegment() = %q, want %q", got, want)
	}
}

func TestPromptSegment_DefaultNamespace(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "1")
	t.Setenv("YAKS_CONTEXT", "dev")
	t.Setenv("YAKS_NAMESPACE", "")

	got := PromptSegment()
	want := "[dev|default]"
	if got != want {
		t.Errorf("PromptSegment() = %q, want %q", got, want)
	}
}

func TestPromptSegment_NoContext(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "1")
	t.Setenv("YAKS_CONTEXT", "")

	if got := PromptSegment(); got != "" {
		t.Errorf("PromptSegment() = %q, want empty when no context", got)
	}
}

func TestPromptSegment_NoPrompt(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "1")
	t.Setenv("YAKS_CONTEXT", "prod")
	t.Setenv("YAKS_NAMESPACE", "default")
	t.Setenv("YAKS_NO_PROMPT", "1")

	if got := PromptSegment(); got != "" {
		t.Errorf("PromptSegment() = %q, want empty when YAKS_NO_PROMPT=1", got)
	}
	if got := PromptSegmentColored(); got != "" {
		t.Errorf("PromptSegmentColored() = %q, want empty when YAKS_NO_PROMPT=1", got)
	}
	if got := ZshPrompt(); got != "" {
		t.Errorf("ZshPrompt() = %q, want empty when YAKS_NO_PROMPT=1", got)
	}
	if got := BashPrompt(); got != "" {
		t.Errorf("BashPrompt() = %q, want empty when YAKS_NO_PROMPT=1", got)
	}
}

func TestPromptSegmentColored_Inactive(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "")
	if got := PromptSegmentColored(); got != "" {
		t.Errorf("PromptSegmentColored() = %q, want empty when inactive", got)
	}
}

func TestPromptSegmentColored_Active(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "1")
	t.Setenv("YAKS_CONTEXT", "prod")
	t.Setenv("YAKS_NAMESPACE", "kube-system")

	got := PromptSegmentColored()
	if !strings.Contains(got, "prod") {
		t.Errorf("PromptSegmentColored() missing context name")
	}
	if !strings.Contains(got, "kube-system") {
		t.Errorf("PromptSegmentColored() missing namespace")
	}
}

func TestZshPrompt_Inactive(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "")
	if got := ZshPrompt(); got != "" {
		t.Errorf("ZshPrompt() = %q, want empty when inactive", got)
	}
}

func TestZshPrompt_Active(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "1")
	t.Setenv("YAKS_CONTEXT", "staging")
	t.Setenv("YAKS_NAMESPACE", "apps")

	got := ZshPrompt()
	if !strings.Contains(got, "staging") || !strings.Contains(got, "apps") {
		t.Errorf("ZshPrompt() = %q, missing context/namespace", got)
	}
	if !strings.Contains(got, "%F{cyan}") || !strings.Contains(got, "%F{yellow}") {
		t.Errorf("ZshPrompt() missing zsh color codes")
	}
}

func TestBashPrompt_Inactive(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "")
	if got := BashPrompt(); got != "" {
		t.Errorf("BashPrompt() = %q, want empty when inactive", got)
	}
}

func TestBashPrompt_Active(t *testing.T) {
	t.Setenv("YAKS_ACTIVE", "1")
	t.Setenv("YAKS_CONTEXT", "prod")
	t.Setenv("YAKS_NAMESPACE", "default")

	got := BashPrompt()
	if !strings.Contains(got, "prod") || !strings.Contains(got, "default") {
		t.Errorf("BashPrompt() missing context/namespace")
	}
}

func TestFishPrompt(t *testing.T) {
	got := FishPrompt()
	if !strings.Contains(got, "YAKS_ACTIVE") {
		t.Errorf("FishPrompt() missing YAKS_ACTIVE check")
	}
	if !strings.Contains(got, "set_color cyan") {
		t.Errorf("FishPrompt() missing cyan color")
	}
	if !strings.Contains(got, "YAKS_CONTEXT") {
		t.Errorf("FishPrompt() missing YAKS_CONTEXT")
	}
	if !strings.Contains(got, "YAKS_NAMESPACE") {
		t.Errorf("FishPrompt() missing YAKS_NAMESPACE")
	}
}

func TestShellInit_Bash(t *testing.T) {
	got := ShellInit("bash")
	if got == "" {
		t.Fatal("ShellInit(bash) returned empty string")
	}
	if !strings.Contains(got, "__yaks_prompt_command") {
		t.Error("ShellInit(bash) missing __yaks_prompt_command function")
	}
	if !strings.Contains(got, "YAKS_ACTIVE") {
		t.Error("ShellInit(bash) missing YAKS_ACTIVE check")
	}
	if !strings.Contains(got, "--shell-eval bash") {
		t.Error("ShellInit(bash) missing --shell-eval wrapper")
	}
}

func TestShellInit_Zsh(t *testing.T) {
	got := ShellInit("zsh")
	if got == "" {
		t.Fatal("ShellInit(zsh) returned empty string")
	}
	if !strings.Contains(got, "__yaks_update_prompt") {
		t.Error("ShellInit(zsh) missing __yaks_update_prompt function")
	}
	if !strings.Contains(got, "PROMPT") {
		t.Error("ShellInit(zsh) missing PROMPT variable")
	}
	if !strings.Contains(got, "--shell-eval zsh") {
		t.Error("ShellInit(zsh) missing --shell-eval wrapper")
	}
}

func TestShellInit_Fish(t *testing.T) {
	got := ShellInit("fish")
	if got == "" {
		t.Fatal("ShellInit(fish) returned empty string")
	}
	if !strings.Contains(got, "__yaks_ps1") {
		t.Error("ShellInit(fish) missing __yaks_ps1 function")
	}
	if !strings.Contains(got, "fish_prompt") {
		t.Error("ShellInit(fish) missing fish_prompt")
	}
	if !strings.Contains(got, "--shell-eval fish") {
		t.Error("ShellInit(fish) missing --shell-eval wrapper")
	}
}

func TestShellInit_Unsupported(t *testing.T) {
	got := ShellInit("tcsh")
	if got != "" {
		t.Errorf("ShellInit(tcsh) = %q, want empty for unsupported shell", got)
	}
}

func TestShellInit_PowerShell(t *testing.T) {
	got := ShellInit("powershell")
	if got == "" {
		t.Fatal("ShellInit(powershell) returned empty string")
	}
	if !strings.Contains(got, "$env:YAKS_ACTIVE") {
		t.Error("ShellInit(powershell) missing YAKS_ACTIVE check")
	}
	if !strings.Contains(got, "--shell-eval powershell") {
		t.Error("ShellInit(powershell) missing --shell-eval wrapper")
	}
	if !strings.Contains(got, "Invoke-Expression") {
		t.Error("ShellInit(powershell) missing Invoke-Expression")
	}
	if !strings.Contains(got, "function prompt") {
		t.Error("ShellInit(powershell) missing prompt function")
	}
}

func TestPowerShellModuleManifest(t *testing.T) {
	got := PowerShellModuleManifest()
	if got == "" {
		t.Fatal("PowerShellModuleManifest() returned empty string")
	}
	if !strings.Contains(got, "RootModule") {
		t.Error("manifest missing RootModule")
	}
	if !strings.Contains(got, "YaksInit.psm1") {
		t.Error("manifest missing reference to YaksInit.psm1")
	}
	if !strings.Contains(got, "FunctionsToExport") {
		t.Error("manifest missing FunctionsToExport")
	}
	if !strings.Contains(got, "'yaks'") {
		t.Error("manifest missing 'yaks' in FunctionsToExport")
	}
	if !strings.Contains(got, "'prompt'") {
		t.Error("manifest missing 'prompt' in FunctionsToExport")
	}
	if !strings.Contains(got, "AliasesToExport") {
		t.Error("manifest missing AliasesToExport")
	}
	if !strings.Contains(got, "'ktx'") {
		t.Error("manifest missing 'ktx' alias")
	}
	if !strings.Contains(got, "'kns'") {
		t.Error("manifest missing 'kns' alias")
	}
	if !strings.Contains(got, "ModuleVersion") {
		t.Error("manifest missing ModuleVersion")
	}
}

func TestPowerShellModuleScript(t *testing.T) {
	got := PowerShellModuleScript()
	if got == "" {
		t.Fatal("PowerShellModuleScript() returned empty string")
	}

	// Should have the prompt function
	if !strings.Contains(got, "function prompt") {
		t.Error("module script missing prompt function")
	}

	// Should have the yaks wrapper function
	if !strings.Contains(got, "function yaks") {
		t.Error("module script missing yaks wrapper function")
	}

	// Should reference --shell-eval powershell
	if !strings.Contains(got, "--shell-eval powershell") {
		t.Error("module script missing --shell-eval powershell")
	}

	// Should have Invoke-Expression for eval
	if !strings.Contains(got, "Invoke-Expression") {
		t.Error("module script missing Invoke-Expression")
	}

	// Should have YAKS_ACTIVE check
	if !strings.Contains(got, "$env:YAKS_ACTIVE") {
		t.Error("module script missing YAKS_ACTIVE check")
	}

	// Should define aliases
	if !strings.Contains(got, "Set-Alias") {
		t.Error("module script missing Set-Alias")
	}
	if !strings.Contains(got, "ktx") {
		t.Error("module script missing ktx alias")
	}
	if !strings.Contains(got, "kns") {
		t.Error("module script missing kns alias")
	}

	// Should have Export-ModuleMember
	if !strings.Contains(got, "Export-ModuleMember") {
		t.Error("module script missing Export-ModuleMember")
	}

	// Should register argument completers
	if !strings.Contains(got, "Register-ArgumentCompleter") {
		t.Error("module script missing Register-ArgumentCompleter")
	}

	// Should handle YAKS_TMPDIR cleanup
	if !strings.Contains(got, "YAKS_TMPDIR") {
		t.Error("module script missing YAKS_TMPDIR cleanup")
	}
}
