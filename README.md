# yaks — Yet Another Kontext Switcher


A multiplatform Kubernetes context and namespace switcher written in Go.  it was very heavily inspired by [kubie](https://github.com/sbstp/kubie)

> **⚠️ DISCLAIMER ⚠️**
>
> This project is in no way meant to be a replacement for [kubie](https://github.com/sbstp/kubie). No claims are made to the great ideas implemented in the OG [kubie](https://github.com/sbstp/kubie). I simply wanted a Windows-capable version. If you are able, you really should probably stick to it. Using this cheap knockoff will probably delete all your pods, make your nodes NotReady and may even cause warts.

yaks spawns **isolated sub-shells** with per-session kubeconfig files so context/namespace changes don't leak between terminals.

## Features

- **Context switching** — spawn a new shell scoped to a single context
- **Namespace switching** — change namespaces within the current context
- **Isolated sessions** — each shell gets its own temporary kubeconfig
- **Interactive selection** — uses [fzf](https://github.com/junegunn/fzf) when available, falls back to numbered list
- **Shell prompt integration** — bash, zsh, and fish support
- **Nested shells** — depth tracking for nested context sessions
- **Multi-kubeconfig** — merges all files from `KUBECONFIG` env var
- **Pre/post/exit hooks** — run commands on context enter, after shell spawn, and on exit
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

Add yaks context info to your shell prompt:

**Bash** (`~/.bashrc`):
```bash
eval "$(yaks init bash)"
```

**Zsh** (`~/.zshrc`):
```bash
eval "$(yaks init zsh)"
```

**Fish** (`~/.config/fish/config.fish`):
```fish
yaks init fish | source
```

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
  # Run before the sub-shell spawns
  pre:
    - name: "warn-prod"
      match: "prod-*"           # glob pattern on context name; omit to match all
      command: "echo '⚠️  PRODUCTION CONTEXT'"

  # Run after the sub-shell spawns (before you get the prompt)
  post:
    - name: "prod-bg"
      match: "prod-*"
      command: "printf '\e]11;#3a0000\a'"   # dark red background
    - name: "set-aws"
      match: "aws-*"
      command: "export AWS_PROFILE=my-profile"

  # Run after the sub-shell exits (cleanup)
  exit:
    - name: "reset-bg"
      command: "printf '\e]11;#000000\a'"   # reset to black
```

Hooks receive the full yaks environment (`YAKS_CONTEXT`, `YAKS_NAMESPACE`, etc.) and are executed through your `$SHELL`, so shell functions and aliases work normally.

A hook failure prints a warning but does **not** abort the context switch or other hooks.

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

### Exit a yaks shell

Simply type `exit` or press `Ctrl+D` to return to the previous shell.

## How it works

1. When you run `yaks ctx <context>`, it:
   - Loads and merges all kubeconfig files from `KUBECONFIG`
   - Creates a **temporary kubeconfig** with only the selected context
   - Spawns a new sub-shell with `KUBECONFIG` pointing to the temp file
   - Sets `YAKS_ACTIVE`, `YAKS_CONTEXT`, `YAKS_NAMESPACE`, and `YAKS_DEPTH` environment variables

2. When you run `yaks ns <namespace>`, it:
   - Updates the namespace in the current kubeconfig (isolated if in a yaks shell)
   - Updates the `YAKS_NAMESPACE` environment variable

3. Changes are **completely isolated** — switching context in one terminal doesn't affect any other terminal.

## Environment variables

### Set by yaks (inside a managed shell)

| Variable | Description |
|---|---|
| `KUBECONFIG` | Path to the temporary isolated kubeconfig created for this shell session |
| `YAKS_ACTIVE` | Set to `1` when inside a yaks-managed shell; unset otherwise |
| `YAKS_CONTEXT` | Name of the active Kubernetes context |
| `YAKS_NAMESPACE` | Name of the active namespace |
| `YAKS_DEPTH` | Nesting depth of yaks shells (`1` for the first, `2` if you `yaks ctx` again inside, etc.) |\n| `YAKS_KUBECONFIG` | Original `KUBECONFIG` path(s) preserved for nested context switching |

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
│   ├── ctx.go                # Context switching command
│   ├── ns.go                 # Namespace switching command
│   ├── info.go               # Status/info display
│   ├── list.go               # List contexts/namespaces
│   ├── init.go               # Shell prompt integration
│   ├── version.go            # Version command
│   └── completion.go         # Shell completion generation
├── pkg/
│   ├── kubeconfig/           # Kubeconfig parsing, loading, merging, saving
│   ├── shell/                # Sub-shell spawning with isolated config
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
