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
	default:
		return ""
	}
}

func bashInit() string {
	return `# yaks shell integration for bash
# Add this to your ~/.bashrc:
#   eval "$(yaks init bash)"

__yaks_ps1() {
    if [ -n "$YAKS_ACTIVE" ] && [ -z "$YAKS_NO_PROMPT" ]; then
        echo -n "[\[\033[1;36m\]${YAKS_CONTEXT}\[\033[0m\]|\[\033[1;33m\]${YAKS_NAMESPACE}\[\033[0m\]] "
    fi
}

if [ -n "$YAKS_ACTIVE" ] && [ -z "$YAKS_NO_PROMPT" ]; then
    PS1="$(__yaks_ps1)${PS1}"
fi
`
}

func zshInit() string {
	return `# yaks shell integration for zsh
# Add this to your ~/.zshrc:
#   eval "$(yaks init zsh)"

__yaks_ps1() {
    if [[ -n "$YAKS_ACTIVE" ]] && [[ -z "$YAKS_NO_PROMPT" ]]; then
        echo -n "[%F{cyan}${YAKS_CONTEXT}%f|%F{yellow}${YAKS_NAMESPACE}%f] "
    fi
}

if [[ -n "$YAKS_ACTIVE" ]] && [[ -z "$YAKS_NO_PROMPT" ]]; then
    PROMPT="$(__yaks_ps1)${PROMPT}"
fi
`
}

func fishInit() string {
	return `# yaks shell integration for fish
# Add this to your ~/.config/fish/config.fish:
#   yaks init fish | source

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

# Prepend to fish_prompt if yaks is active
if set -q YAKS_ACTIVE; and not set -q YAKS_NO_PROMPT
    functions -c fish_prompt __yaks_original_prompt
    function fish_prompt
        __yaks_ps1
        __yaks_original_prompt
    end
end
`
}
