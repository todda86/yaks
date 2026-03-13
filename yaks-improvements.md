# yaks — Proposed Improvements

Lessons learned from building the Tanium enterprise deployment package.
These changes would simplify shell integration across all platforms.

---

## 1. `yaks activate` command

**Problem:** Starting a yaks session at shell startup requires shelling out
to kubectl to discover the current context, then passing it to `yaks ctx`:

```bash
eval "$(yaks ctx $(kubectl config current-context) --shell-eval bash)"
```

**Proposal:** A dedicated `yaks activate` that reads the current kubeconfig
context internally and emits the shell-eval output:

```bash
# bash/zsh
eval "$(yaks activate --shell-eval bash)"

# fish
yaks activate --shell-eval fish | source

# powershell
yaks activate --shell-eval powershell | Out-String | Invoke-Expression
```

**Impact:** High — simplifies every shell's init, removes kubectl dependency
at startup.  
**Effort:** Low — yaks already reads kubeconfig; just wire it up as a new
subcommand (or flag on `ctx`).  
**Platforms:** All

---

## 2. `--quiet` flag on `--shell-eval`

**Problem:** `yaks ctx --shell-eval powershell prod` writes env-var
assignments to stdout and a status line (`prod|default — switched`) to
stderr. In PowerShell, `2>&1` is a common idiom that merges both streams,
causing `Invoke-Expression` to try to execute the status text as code.

**Proposal:** A `--quiet` / `--no-status` flag that suppresses the stderr
status message when called programmatically:

```powershell
$output = & yaks ctx --shell-eval powershell --quiet @remaining
$output | Out-String | Invoke-Expression
```

**Impact:** Medium — prevents a class of subtle eval bugs in wrappers.  
**Effort:** Low — guard the status fprintf behind a flag check.  
**Platforms:** All (though the bug primarily bites PowerShell)

---

## 3. `yaks init powershell --module`

**Problem:** `yaks init powershell | Invoke-Expression` creates wrapper
functions in whatever **scope** calls it. Inside a PowerShell module
(`.psm1`), those functions are trapped in module scope and invisible to the
user's session. Enterprise deployments need a proper module with
`FunctionsToExport` for auto-discovery.

**Proposal:** `yaks init powershell --module` emits (or installs) a proper
PowerShell module:

```
YaksInit/
  YaksInit.psd1   # manifest with FunctionsToExport = @('yaks','ktx','kns')
  YaksInit.psm1   # module script — wrapper, aliases, prompt, completions
```

Could support `--install` to write directly to the modules path:

```powershell
yaks init powershell --module --install
# → installs to C:\Program Files\PowerShell\Modules\YaksInit\
```

**Impact:** High — eliminates all scope issues for managed deployments.  
**Effort:** Medium — need to emit two files with correct manifest structure.  
**Platforms:** PowerShell (Windows, macOS, Linux)

---

## 4. Completion re-registration for wrapper functions

**Problem:** Cobra's `yaks completion powershell` registers a completer for
the `yaks` command name. When a module exports a `yaks` *function* that
shadows the binary, the completer registration works but is fragile — it
depends on load order and scope.

**Proposal:** The completion script could detect whether `yaks` resolves to
a function vs. an application and register appropriately, or the module
output from #3 could handle this automatically.

**Impact:** Low  
**Effort:** Low  
**Platforms:** PowerShell

---

## Priority

| # | Change | Impact | Effort | Platforms |
|---|--------|--------|--------|-----------|
| 1 | `yaks activate --shell-eval` | High | Low | All |
| 2 | `--quiet` on `--shell-eval` | Medium | Low | All |
| 3 | `yaks init powershell --module` | High | Medium | PowerShell |
| 4 | Completion re-registration | Low | Low | PowerShell |
