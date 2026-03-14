# yaks — Yet Another Kontext Switcher


A multiplatform Kubernetes context and namespace switcher written in Go.  it was very heavily inspired by [kubie](https://github.com/sbstp/kubie)

> **⚠️ DISCLAIMER ⚠️**
>
> This project is in no way meant to be a replacement for [kubie](https://github.com/sbstp/kubie). No claims are made to the great ideas implemented in the OG [kubie](https://github.com/sbstp/kubie). I simply wanted a Windows-capable version. If you are able, you really should probably stick to it. Using this cheap knockoff will probably delete all your pods, make your nodes NotReady and may even cause warts.

yaks uses **in-place eval-based switching** (bash, zsh, fish, PowerShell) so context and namespace changes happen in your current shell — no sub-shell nesting. Each session gets its own temporary kubeconfig so changes don't leak between terminals.

## Features

- **Context switching** — switch contexts in-place via shell wrapper functions (no sub-shells)
- **Namespace switching** — change namespaces within the current context, with existence validation
- **Isolated sessions** — each session gets its own temporary kubeconfig
- **Interactive selection** — uses [fzf](https://github.com/junegunn/fzf) when available, falls back to numbered list
- **Shell prompt integration** — bash, zsh, fish, and PowerShell support
- **Multi-kubeconfig** — merges all files from `KUBECONFIG` env var
- **Pre/post/exit hooks** — run commands on context enter/switch with glob matching and first-match `stop` control
- **Cross-platform** — builds for Linux, macOS, and Windows (amd64/arm64)

## Installation

### From source

```bash
go install github.com/todda86/yaks@latest
```

### Build from repo

```bash
git clone https://github.com/todda86/yaks.git
cd yaks
make build
```

### Cross-compile for all platforms

```bash
make cross-compile
# Binaries in dist/
```

## Usage

### Activate a session

The `activate` command reads the current context from your kubeconfig and sets up an isolated yaks session in your current shell — no need to know or pass the context name, and no dependency on `kubectl`:

```bash
# Bash / Zsh
eval "$(yaks activate --shell-eval bash)"
eval "$(yaks activate --shell-eval zsh)"

# Fish
yaks activate --shell-eval fish | source

# PowerShell
yaks activate --shell-eval powershell | Out-String | Invoke-Expression
```

Optionally override the namespace:

```bash
eval "$(yaks activate --shell-eval bash -n kube-system)"
```

This is the recommended way to bootstrap a yaks session at shell startup — add it to your shell RC file alongside `yaks init` (see [Shell prompt integration](#shell-prompt-integration) below).

### Switch context

```bash
# Interactive context selector (uses fzf if available)
yaks ctx

# Switch to a specific context
yaks ctx my-cluster

# Switch to a context with a specific namespace
yaks ctx my-cluster -n kube-system
```

### Switch namespace

```bash
# Interactive namespace selector
yaks ns

# Switch to a specific namespace
yaks ns kube-system
```

### View current status

```bash
yaks info
```

Output:
```
+-----------------------------------------+
|             yaks status                 |
+-----------------------------------------+
  Context:   production
  Namespace: default
  Cluster:   prod-cluster
  Server:    https://k8s.example.com:6443
  User:      admin
  Shell:     active (depth: 1)

  Available contexts:
    > production (default)
      staging (kube-system)
      development
```

### List contexts and namespaces

```bash
# List all contexts
yaks list contexts

# List all namespaces in the current cluster
yaks list namespaces
```

### Shell prompt integration

Add yaks context info to your shell prompt and optionally activate a session at startup:

**Bash** (`~/.bashrc`):
```bash
eval "$(yaks init bash)"
eval "$(yaks activate --shell-eval bash)"   # optional: auto-activate current context
```

**Zsh** (`~/.zshrc`):
```bash
eval "$(yaks init zsh)"
eval "$(yaks activate --shell-eval zsh)"    # optional: auto-activate current context
```

**Fish** (`~/.config/fish/config.fish`):
```fish
yaks init fish | source
yaks activate --shell-eval fish | source    # optional: auto-activate current context
```

**PowerShell** (`$PROFILE`):
```powershell
yaks init powershell | Out-String | Invoke-Expression
yaks activate --shell-eval powershell | Out-String | Invoke-Expression  # optional
```

> **Tip:** `yaks activate` reads the current context directly from your kubeconfig — no `kubectl` dependency. Previously this required: `eval "$(yaks ctx $(kubectl config current-context) --shell-eval bash)"`. The activate command replaces that pattern entirely.

### Shell completions

```bash
# Bash
source <(yaks completion bash)

# Zsh
source <(yaks completion zsh)

# Fish
yaks completion fish | source

# PowerShell
yaks completion powershell | Out-String | Invoke-Expression
```

### Hooks

yaks supports pre, post, and exit hooks that run shell commands at context-switch lifecycle points. Create a config file at `~/.config/yaks/config.yaml` (or set `YAKS_CONFIG` to a custom path):

```yaml
hooks:
  # Run before the context switch
  pre:
    - name: "warn-prod"
      match: "prod-*"           # glob pattern on context name; omit to match all
      command: "echo '⚠️  PRODUCTION CONTEXT'"

  # Run after the context switch
  post:
    - name: "prod-bg"
      match: "prod-*"
      command: "printf '\e]11;#3a0000\a'"   # dark red background
      stop: true                             # don't evaluate further hooks
    - name: "default-bg"
      match: "*"
      command: "printf '\e]11;#000000\a'"   # default black background

  # Run on exit (cleanup)
  exit:
    - name: "reset-bg"
      command: "printf '\e]11;#000000\a'"   # reset to black
```

#### Hook fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | A descriptive label for the hook |
| `match` | string | Glob pattern matched against the context name. Omit or leave empty to match all contexts |
| `command` | string | Shell command to execute |
| `stop` | bool | When `true`, no further hooks in the list are evaluated after this one matches |

Hooks receive the full yaks environment (`YAKS_CONTEXT`, `YAKS_NAMESPACE`, etc.) and are executed through your `$SHELL`, so shell functions and aliases work normally.

A hook failure prints a warning but does **not** abort the context switch or other hooks.

> **Tip:** Use `stop: true` on specific patterns before a wildcard catch-all to implement first-match-wins behavior (e.g., set a red background for prod contexts, black for everything else).

#### Example: terminal background color per context (fish)

Add a fish function to `~/.config/fish/functions/yaks_set_bg.fish`:

```fish
function yaks_set_bg
    switch $YAKS_CONTEXT
        case 'prod-*'
            printf '\e]11;#3a0000\a'   # dark red
        case 'staging-*'
            printf '\e]11;#003a00\a'   # dark green
        case '*'
            printf '\e]11;#000000\a'   # default black
    end
end
```

Then in `~/.config/yaks/config.yaml`:

```yaml
hooks:
  post:
    - name: "set-terminal-bg"
      command: "yaks_set_bg"
  exit:
    - name: "reset-terminal-bg"
      command: "printf '\e]11;#000000\a'"
```

Now every time you enter a prod context, your terminal goes red. When you `exit`, it resets.

## How it works

1. When you run `yaks activate --shell-eval <shell>` (typically in your shell RC):
   - yaks reads the current context from the merged kubeconfig
   - Creates a **temporary kubeconfig** with only that context
   - Outputs shell-specific `export`/`set` commands to stdout
   - Your shell **evals** the output, setting env vars in the current session
   - Runs any matching pre/post hooks
   - No `kubectl` dependency — all context discovery is done internally

2. When you run `yaks ctx <context>` (with shell init sourced):
   - The shell wrapper function calls `yaks ctx --shell-eval <shell> <context>`
   - yaks loads and merges all kubeconfig files from `KUBECONFIG`
   - Creates a **temporary kubeconfig** with only the selected context
   - Outputs shell-specific `export`/`set` commands to stdout
   - The wrapper function **evals** the output, setting env vars in your current shell — no sub-shell spawned
   - Runs any matching pre/post hooks

3. When you run `yaks ns <namespace>`:
   - Validates the namespace exists in the cluster (via `kubectl`)
   - Updates the namespace in the current kubeconfig
   - Sets `YAKS_NAMESPACE` in the current shell (via eval, same as `ctx`)

4. Changes are **completely isolated** — switching context in one terminal doesn't affect any other terminal.

> **Note:** Without `yaks init` sourced, `yaks ctx` falls back to spawning a sub-shell. The eval-based approach is recommended as it avoids shell nesting.

## Environment variables

### Set by yaks (inside a managed shell)

| Variable | Description |
|---|---|
| `KUBECONFIG` | Path to the temporary isolated kubeconfig created for this shell session |
| `YAKS_ACTIVE` | Set to `1` when inside a yaks-managed shell; unset otherwise |
| `YAKS_CONTEXT` | Name of the active Kubernetes context |
| `YAKS_NAMESPACE` | Name of the active namespace |
| `YAKS_TMPDIR` | Path to the temporary directory holding the isolated kubeconfig for this session |
| `YAKS_KUBECONFIG` | Original `KUBECONFIG` path(s) preserved for context switching |

### User-configurable

| Variable | Description |
|---|---|
| `YAKS_CONFIG` | Override the hooks config file path (default: `~/.config/yaks/config.yaml`, or `$XDG_CONFIG_HOME/yaks/config.yaml`) |
| `YAKS_SILENT` | Set to `1` to suppress status messages when switching context/namespace |
| `YAKS_NO_PROMPT` | Set to `1` to suppress the shell prompt segment (the `[context|namespace]` prefix) even when `yaks init` is sourced |
| `SHELL` | Used to detect which shell to spawn and which shell runs hook commands (falls back to `/bin/zsh`, `/bin/bash`, `/bin/sh`) |

## Project structure

```
yaks/
├── main.go                    # Entry point
├── cmd/                       # CLI commands (cobra)
│   ├── root.go               # Root command & subcommand registration
│   ├── activate.go           # Activate session from current kubeconfig context
│   ├── ctx.go                # Context switching command
│   ├── ns.go                 # Namespace switching command
│   ├── info.go               # Status/info display
│   ├── list.go               # List contexts/namespaces
│   ├── init.go               # Shell prompt integration
│   ├── version.go            # Version command
│   └── completion.go         # Shell completion generation
├── pkg/
│   ├── kubeconfig/           # Kubeconfig parsing, loading, merging, saving
│   ├── shell/                # Shell integration & isolated kubeconfig setup
│   ├── hooks/                # Pre/post/exit hook config and execution
│   ├── state/                # Environment-based state management
│   ├── prompt/               # Shell prompt integration scripts
│   └── fzf/                  # Interactive selection (fzf + fallback)
├── Makefile                  # Build, test, cross-compile targets
├── .goreleaser.yml           # Release automation config
└── .gitignore
```

## License

MIT
