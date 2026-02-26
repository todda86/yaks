# yaks — Possible Features

## Multi-Cluster Operations

- **`exec --all` / `exec --glob`** — Run the same command across multiple contexts (e.g., `yaks exec 'prod-*' kube-system -- kubectl get nodes`), with parallel execution and aggregated output.
- **Context groups** — Define named groups (e.g., `production`, `staging`) and target them with exec or switch between them.

## Productivity

- **Context aliases** — Short names for long context strings (`yaks alias p prod-us-east-1-eks-cluster`), so `yaks ctx p` just works.
- **Bookmarks** — Save context+namespace combos as named shortcuts (`yaks bookmark save my-app prod monitoring`).
- **Recent contexts** — `yaks recent` to show/switch to your last N context+namespace combos, ordered by recency.
- **Namespace caching** — Cache namespace lists per context with a TTL so switching is instant instead of hitting the API every time.

## Safety & Ops

- **Context health check** — `yaks ping` or `yaks health` to quickly test connectivity to one or all clusters.
- **Hooks** — ✅ **IMPLEMENTED** — Pre/post/exit switch hooks via `~/.config/yaks/config.yaml` with glob pattern matching on context names.
- **Audit log** — Append context switches to `~/.kube/yaks-audit.log` with timestamps for compliance/debugging.
- **Confirmation for prod** — Configurable prompt like "You are switching to a production context. Continue?" based on name patterns.

## Config Management

- **`yaks lint`** — Validate kubeconfig for common issues (duplicate contexts, missing certs, unreachable servers).
- **`yaks edit`** — Open context/cluster config in `$EDITOR`.
- **`yaks delete`** — Remove a context from kubeconfig.
- **`yaks import`** — Merge a new kubeconfig file into the default one.
- **Config file** (`~/.config/yaks/config.yaml`) — Persistent settings for aliases, groups, hooks, cache TTL, prod-warning patterns.

## Visibility

- **`info` subcommands** — `yaks info ctx`, `yaks info ns`, `yaks info depth` for scriptable single-value output.
- **Cluster resource summary** — `yaks status` showing node count, pod count, resource usage at a glance.
