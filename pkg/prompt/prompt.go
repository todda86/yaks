package prompt

import (
	"fmt"
	"strings"

	"github.com/todda86/yaks/pkg/state"
)

// PromptSegment returns a formatted string suitable for embedding in a shell prompt.
// The format is: [context|namespace]
func PromptSegment() string {
	if !state.IsActive() || state.NoPrompt() {
		return ""
	}

	ctx := state.GetCurrentContext()
	ns := state.GetCurrentNamespace()

	if ctx == "" {
		return ""
	}

	if ns == "" {
		ns = "default"
	}

	return fmt.Sprintf("[%s|%s]", ctx, ns)
}

// PromptSegmentColored returns a colored prompt segment for terminals that support ANSI colors.
func PromptSegmentColored() string {
	if !state.IsActive() || state.NoPrompt() {
		return ""
	}

	ctx := state.GetCurrentContext()
	ns := state.GetCurrentNamespace()

	if ctx == "" {
		return ""
	}

	if ns == "" {
		ns = "default"
	}

	// Cyan for context, yellow for namespace
	return fmt.Sprintf("\033[1;36m%s\033[0m|\033[1;33m%s\033[0m", ctx, ns)
}

// ZshPrompt returns a zsh-compatible prompt segment using %F{} color codes.
func ZshPrompt() string {
	if !state.IsActive() || state.NoPrompt() {
		return ""
	}

	ctx := state.GetCurrentContext()
	ns := state.GetCurrentNamespace()

	if ctx == "" {
		return ""
	}

	if ns == "" {
		ns = "default"
	}

	return fmt.Sprintf("%%F{cyan}%s%%f|%%F{yellow}%s%%f", ctx, ns)
}

// FishPrompt returns a fish-compatible prompt function body.
func FishPrompt() string {
	return strings.TrimSpace(`
if set -q YAKS_ACTIVE; and not set -q YAKS_NO_PROMPT
    set_color cyan
    echo -n $YAKS_CONTEXT
    set_color normal
    echo -n '|'
    set_color yellow
    echo -n $YAKS_NAMESPACE
    set_color normal
    echo -n ' '
end
`)
}

// BashPrompt returns a bash PS1 segment.
func BashPrompt() string {
	if !state.IsActive() || state.NoPrompt() {
		return ""
	}

	ctx := state.GetCurrentContext()
	ns := state.GetCurrentNamespace()

	if ctx == "" {
		return ""
	}

	if ns == "" {
		ns = "default"
	}

	return fmt.Sprintf("\\[\\033[1;36m\\]%s\\[\\033[0m\\]|\\[\\033[1;33m\\]%s\\[\\033[0m\\]", ctx, ns)
}

// ShellInit returns a shell initialization script for the given shell type.
func ShellInit(shellType string) string {
	switch shellType {
	case "bash":
		return bashInit()
	case "zsh":
		return zshInit()
	case "fish":
		return fishInit()
	case "powershell":
		return powershellInit()
	default:
		return ""
	}
}

func bashInit() string {
	return `# yaks shell integration for bash
# Add this to your ~/.bashrc:
#   eval "$(yaks init bash)"

# Dynamic prompt: shows [context|namespace] when yaks is active
__yaks_orig_ps1="$PS1"
__yaks_prompt_command() {
    if [ -n "$YAKS_ACTIVE" ] && [ -z "$YAKS_NO_PROMPT" ]; then
        PS1="[\[\033[1;36m\]${YAKS_CONTEXT}\[\033[0m\]|\[\033[1;33m\]${YAKS_NAMESPACE}\[\033[0m\]] ${__yaks_orig_ps1}"
    else
        PS1="$__yaks_orig_ps1"
    fi
}
PROMPT_COMMAND="__yaks_prompt_command${PROMPT_COMMAND:+;$PROMPT_COMMAND}"

# Shell wrapper: intercepts ctx/ns to eval env changes in the current shell
yaks() {
    case "${1:-}" in
        ctx|context)
            local _yaks_old_tmpdir="${YAKS_TMPDIR:-}"
            local _yaks_eval
            _yaks_eval=$(command yaks ctx --shell-eval bash "${@:2}")
            local _yaks_status=$?
            if [ $_yaks_status -eq 0 ] && [ -n "$_yaks_eval" ]; then
                eval "$_yaks_eval"
                if [ -n "$_yaks_old_tmpdir" ] && [ "$_yaks_old_tmpdir" != "${YAKS_TMPDIR:-}" ]; then
                    rm -rf "$_yaks_old_tmpdir" 2>/dev/null
                fi
            fi
            return $_yaks_status
            ;;
        ns|namespace)
            local _yaks_eval
            _yaks_eval=$(command yaks ns --shell-eval bash "${@:2}")
            local _yaks_status=$?
            if [ $_yaks_status -eq 0 ] && [ -n "$_yaks_eval" ]; then
                eval "$_yaks_eval"
            fi
            return $_yaks_status
            ;;
        *)
            command yaks "$@"
            ;;
    esac
}
`
}

func zshInit() string {
	return `# yaks shell integration for zsh
# Add this to your ~/.zshrc:
#   eval "$(yaks init zsh)"

# Dynamic prompt: shows [context|namespace] when yaks is active
__yaks_orig_ps1="$PROMPT"
precmd_functions+=(__yaks_update_prompt)
__yaks_update_prompt() {
    if [[ -n "$YAKS_ACTIVE" ]] && [[ -z "$YAKS_NO_PROMPT" ]]; then
        PROMPT="[%F{cyan}${YAKS_CONTEXT}%f|%F{yellow}${YAKS_NAMESPACE}%f] ${__yaks_orig_ps1}"
    else
        PROMPT="$__yaks_orig_ps1"
    fi
}

# Shell wrapper: intercepts ctx/ns to eval env changes in the current shell
yaks() {
    case "${1:-}" in
        ctx|context)
            local _yaks_old_tmpdir="${YAKS_TMPDIR:-}"
            local _yaks_eval
            _yaks_eval=$(command yaks ctx --shell-eval zsh "${@:2}")
            local _yaks_status=$?
            if (( _yaks_status == 0 )) && [[ -n "$_yaks_eval" ]]; then
                eval "$_yaks_eval"
                if [[ -n "$_yaks_old_tmpdir" ]] && [[ "$_yaks_old_tmpdir" != "${YAKS_TMPDIR:-}" ]]; then
                    rm -rf "$_yaks_old_tmpdir" 2>/dev/null
                fi
            fi
            return $_yaks_status
            ;;
        ns|namespace)
            local _yaks_eval
            _yaks_eval=$(command yaks ns --shell-eval zsh "${@:2}")
            local _yaks_status=$?
            if (( _yaks_status == 0 )) && [[ -n "$_yaks_eval" ]]; then
                eval "$_yaks_eval"
            fi
            return $_yaks_status
            ;;
        *)
            command yaks "$@"
            ;;
    esac
}
`
}

func fishInit() string {
	return `# yaks shell integration for fish
# Add this to your ~/.config/fish/config.fish:
#   yaks init fish | source

# Dynamic prompt: shows context|namespace when yaks is active
function __yaks_ps1
    if set -q YAKS_ACTIVE; and not set -q YAKS_NO_PROMPT
        set_color cyan
        echo -n $YAKS_CONTEXT
        set_color normal
        echo -n '|'
        set_color yellow
        echo -n $YAKS_NAMESPACE
        set_color normal
        echo -n ' '
    end
end

# Always wrap fish_prompt to conditionally show yaks info
if not functions -q __yaks_original_prompt
    functions -c fish_prompt __yaks_original_prompt
end
function fish_prompt
    __yaks_ps1
    __yaks_original_prompt
end

# Shell wrapper: intercepts ctx/ns to eval env changes in the current shell
function yaks --wraps=yaks --description 'yaks context/namespace switcher'
    if test (count $argv) -ge 1
        switch $argv[1]
            case ctx context
                set -l _yaks_old_tmpdir $YAKS_TMPDIR
                command yaks ctx --shell-eval fish $argv[2..] | source
                set -l _yaks_status $pipestatus[1]
                if test $_yaks_status -eq 0 -a -n "$_yaks_old_tmpdir" -a "$_yaks_old_tmpdir" != "$YAKS_TMPDIR"
                    command rm -rf $_yaks_old_tmpdir 2>/dev/null
                end
                return $_yaks_status
            case ns namespace
                command yaks ns --shell-eval fish $argv[2..] | source
                return $pipestatus[1]
            case '*'
                command yaks $argv
        end
    else
        command yaks $argv
    end
end
`
}

// psCompleterBlock returns the PowerShell completer scriptblock and
// Register-ArgumentCompleter calls. Shared by inline init and module.
// Uses [char]96 for backtick and [char]9 for tab to avoid Go raw string conflicts.
func psCompleterBlock() string {
	return `
# ---------------------------------------------------------------------------
# Tab completion — Register-ArgumentCompleter -Native works for both
# native commands AND functions, fires for every positional argument,
# and runs in caller scope (avoids module scope isolation issues).
# Follows Cobra's proven PowerShell completion pattern.
# ---------------------------------------------------------------------------
filter __yaks_escapeStringWithSpecialChars {
    $bt = [char]96
    $_ -replace ('\s|#|@|\$|;|,|''|\{|\}|\(|\)|"|' + $bt + '|\||<|>|&'), ($bt + '$&')
}

$__yaksCompleterBlock = {
    param($wordToComplete, $commandAst, $cursorPosition)
    $yaksBin = (Get-Command yaks -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1).Source
    if (-not $yaksBin) { return }

    # Map alias commands to their yaks subcommand
    $cmdName = $commandAst.CommandElements[0].ToString()
    $completeArgs = @()
    switch ($cmdName) {
        'ktx' { $completeArgs += 'ctx' }
        'kns' { $completeArgs += 'ns' }
    }
    for ($i = 1; $i -lt $commandAst.CommandElements.Count; $i++) {
        $completeArgs += $commandAst.CommandElements[$i].ToString()
    }
    if ($wordToComplete -eq '') { $completeArgs += '' }

    # Call __complete and collect all output
    $out = @(& $yaksBin __complete @completeArgs 2>$null)
    if ($out.Count -eq 0) { return }

    # Directive is the last line (e.g. ":4")
    [int]$directive = 0
    if ($out[-1] -match '^:(\d+)$') { $directive = [int]$Matches[1] }
    $out = $out | Where-Object { $_ -notmatch '^:' }

    # ShellCompDirectiveError - abort
    if (($directive -band 1) -ne 0) { return }

    # Parse completions into name/description pairs
    [Array]$values = $out | ForEach-Object {
        $n, $d = $_ -split ([char]9), 2
        if (-not $d) { $d = ' ' }
        [PSCustomObject]@{ Name = $n; Description = $d }
    }

    # Trailing space unless NoSpace directive
    $space = ' '
    if (($directive -band 2) -ne 0) { $space = '' }

    # Filter by word being completed
    $values = $values | Where-Object { $_.Name -like "$wordToComplete*" }

    # NoFileComp: emit empty string to suppress path completion when no results
    if (($directive -band 4) -ne 0 -and $values.Length -eq 0) {
        ''
        return
    }

    # Emit CompletionResult objects — escaped name + trailing space works
    # correctly across all PSReadLine modes (Complete, MenuComplete, TabCompleteNext)
    $values | ForEach-Object {
        $ct = ($_.Name | __yaks_escapeStringWithSpecialChars) + $space
        [System.Management.Automation.CompletionResult]::new($ct, $_.Name, 'ParameterValue', $_.Description)
    }
}
Register-ArgumentCompleter -Native -CommandName 'yaks' -ScriptBlock $__yaksCompleterBlock
Register-ArgumentCompleter -Native -CommandName 'ktx'  -ScriptBlock $__yaksCompleterBlock
Register-ArgumentCompleter -Native -CommandName 'kns'  -ScriptBlock $__yaksCompleterBlock
`
}

func powershellInit() string {
	// Simple wrapper functions + Register-ArgumentCompleter -Native for completion.
	// -Native completers run in the caller's scope and work for all positional args.
	return `# yaks shell integration for PowerShell
# Add this to your $PROFILE:
#   yaks init powershell | Out-String | Invoke-Expression

# Dynamic prompt: shows [context|namespace] when yaks is active
$__yaks_orig_prompt = $function:prompt
function prompt {
    $p = & $__yaks_orig_prompt
    if ($env:YAKS_ACTIVE -eq '1' -and -not $env:YAKS_NO_PROMPT) {
        $ctx = $env:YAKS_CONTEXT
        $ns  = $env:YAKS_NAMESPACE
        if ($ctx) {
            Write-Host -NoNewline "[$ctx|$ns] " -ForegroundColor Cyan
        }
    }
    return $p
}

# Shell wrapper: intercepts ctx/ns to eval env changes in the current shell
function yaks {
    if ($args.Count -ge 1) {
        switch ($args[0]) {
            { $_ -in 'ctx','context' } {
                $oldTmpDir = $env:YAKS_TMPDIR
                $remaining = @($args | Select-Object -Skip 1)
                $output = & (Get-Command yaks -CommandType Application | Select-Object -First 1) ctx --shell-eval powershell @remaining
                $exitCode = $LASTEXITCODE
                if ($exitCode -eq 0 -and $output) {
                    $output | Out-String | Invoke-Expression
                    if ($oldTmpDir -and $oldTmpDir -ne $env:YAKS_TMPDIR) {
                        Remove-Item -Recurse -Force $oldTmpDir -ErrorAction SilentlyContinue
                    }
                }
                return
            }
            { $_ -in 'ns','namespace' } {
                $remaining = @($args | Select-Object -Skip 1)
                $output = & (Get-Command yaks -CommandType Application | Select-Object -First 1) ns --shell-eval powershell @remaining
                $exitCode = $LASTEXITCODE
                if ($exitCode -eq 0 -and $output) {
                    $output | Out-String | Invoke-Expression
                }
                return
            }
        }
    }
    & (Get-Command yaks -CommandType Application | Select-Object -First 1) @args
}

function ktx { yaks ctx @args }
function kns { yaks ns @args }
` + psCompleterBlock()
}

// PowerShellModuleManifest returns the content of a YaksInit.psd1 module manifest.
// This manifest declares the module metadata and, critically, uses FunctionsToExport
// to make the wrapper functions visible outside of module scope.
func PowerShellModuleManifest() string {
	return `#
# Module manifest for YaksInit
# Generated by: yaks init powershell --module
#
@{
    RootModule        = 'YaksInit.psm1'
    ModuleVersion     = '1.0.0'
    GUID              = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'
    Author            = 'yaks'
    Description       = 'Shell integration for yaks — Kubernetes context & namespace switcher'
    PowerShellVersion = '5.1'
    FunctionsToExport = @('yaks', 'prompt', 'ktx', 'kns')
    CmdletsToExport   = @()
    VariablesToExport = @()
    AliasesToExport   = @()
}
`
}

// PowerShellModuleScript returns the content of a YaksInit.psm1 module script.
// Uses Register-ArgumentCompleter -Native for reliable tab completion that
// works across all positional args and avoids module scope issues.
// Uses [char]9 (tab) instead of backtick-t so everything fits in Go raw strings.
func PowerShellModuleScript() string {
	return `#
# YaksInit.psm1 — yaks shell integration module
# Generated by: yaks init powershell --module
#
# Install: yaks init powershell --module --install
# Usage:   Import-Module YaksInit
#

# ---------------------------------------------------------------------------
# Preserve the original prompt so we can chain to it
# ---------------------------------------------------------------------------
if (-not (Get-Variable -Name '__yaks_orig_prompt' -Scope Script -ErrorAction SilentlyContinue)) {
    $script:__yaks_orig_prompt = $function:prompt
}

# ---------------------------------------------------------------------------
# Dynamic prompt: shows [context|namespace] when yaks is active
# ---------------------------------------------------------------------------
function prompt {
    $p = if ($script:__yaks_orig_prompt) { & $script:__yaks_orig_prompt } else { "PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) " }
    if ($env:YAKS_ACTIVE -eq '1' -and -not $env:YAKS_NO_PROMPT) {
        $ctx = $env:YAKS_CONTEXT
        $ns  = $env:YAKS_NAMESPACE
        if ($ctx) {
            Write-Host -NoNewline "[$ctx|$ns] " -ForegroundColor Cyan
        }
    }
    return $p
}

# ---------------------------------------------------------------------------
# Shell wrapper: intercepts ctx/ns to eval env changes in the current shell
# ---------------------------------------------------------------------------
function yaks {
    $yaksBin = (Get-Command yaks -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1).Source
    if (-not $yaksBin) {
        Write-Error "yaks binary not found in PATH"
        return
    }

    if ($args.Count -ge 1) {
        switch ($args[0]) {
            { $_ -in 'ctx','context' } {
                $oldTmpDir = $env:YAKS_TMPDIR
                $remaining = @($args | Select-Object -Skip 1)
                $output = & $yaksBin ctx --shell-eval powershell @remaining
                $exitCode = $LASTEXITCODE
                if ($exitCode -eq 0 -and $output) {
                    $output | Out-String | Invoke-Expression
                    if ($oldTmpDir -and $oldTmpDir -ne $env:YAKS_TMPDIR) {
                        Remove-Item -Recurse -Force $oldTmpDir -ErrorAction SilentlyContinue
                    }
                }
                return
            }
            { $_ -in 'ns','namespace' } {
                $remaining = @($args | Select-Object -Skip 1)
                $output = & $yaksBin ns --shell-eval powershell @remaining
                $exitCode = $LASTEXITCODE
                if ($exitCode -eq 0 -and $output) {
                    $output | Out-String | Invoke-Expression
                }
                return
            }
        }
    }
    & $yaksBin @args
}

# ---------------------------------------------------------------------------
# Convenience wrappers: ktx = yaks ctx, kns = yaks ns
# ---------------------------------------------------------------------------
function ktx { yaks ctx @args }
function kns { yaks ns @args }
` + psCompleterBlock() + `
Export-ModuleMember -Function yaks, prompt, ktx, kns
`
}
