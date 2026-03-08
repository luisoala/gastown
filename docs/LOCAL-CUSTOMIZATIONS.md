# Local/Custom Branch — Forensics & Customization Tracker

This document tracks all customizations on the `local/custom` branch that
diverge from upstream `origin/main`. It records rationale, affected files,
side effects, and upgrade considerations for each change.

**Branch**: `local/custom`
**Fork**: `git@github.com:luisoala/gastown.git`
**Last upstream merge**: v0.11.0 (191 commits, `fffae54c`)
**Last updated**: 2026-03-08

---

## 1. Scope EnsureAllMetadata to registered rigs only
**Commit**: `121aa5c0`
**Files**: `internal/beads/metadata.go`
**Problem**: When multiple Gas Towns share the same Dolt data directory,
`EnsureAllMetadata` creates phantom `.beads/` directories for databases
belonging to other towns.
**Fix**: Filter `ListDatabases` output to only process databases registered
in `rigs.json` (plus `hq`).
**Side effects**: Towns without `rigs.json` fall back to old behavior (process
all databases). No regression for single-town setups.
**Upstream candidate**: Yes — guards a real multi-town footgun.

## 2. Town-name-prefixed tmux sessions
**Commit**: `e5cd109a`
**Files**: `internal/session/registry.go`, various session creation paths
**Problem**: Multiple Gas Towns on the same machine had tmux session name
collisions (e.g., two `hq-mayor` sessions).
**Fix**: Prefix all tmux sessions with town name from `town.json`
(e.g., `gt-hq-mayor` → `paper-town-hq-mayor`).
**Side effects**: Existing sessions lose their prefix match on upgrade — daemon
handles migration on restart. Scripts expecting old names need updating.
**Upstream candidate**: Yes — enables multi-town on single machine.

## 3. `gt town list` and `gt town switch`
**Commit**: `61ba9dbd`
**Files**: `cmd/town_list.go`, `cmd/town_switch.go`
**Problem**: No built-in way to discover or navigate between towns.
**Fix**: `gt town list` scans sibling directories for `town.json`.
`gt town switch <name>` validates and switches tmux client.
**Side effects**: None — additive commands only.
**Upstream candidate**: Yes.

## 4. `gt refs` — cross-repo reference pool
**Commit**: `584235eb`
**Files**: `cmd/refs.go`, `cmd/refs_add.go`, `cmd/refs_sync.go`,
`internal/polecat/polecat.md.tmpl`
**Problem**: Polecats needed access to reference repos (e.g., upstream gastown
source) without full clones in each worktree.
**Fix**: Manages `$GT_REF_POOL` directory with shallow clones. Injects pool
info into polecat role templates.
**Side effects**: None if `GT_REF_POOL` unset. Pool repos consume disk.
**Upstream candidate**: Maybe — niche use case.

## 5. Linux credential swap + quota rotation dog
**Commit**: `480703e0`
**Files**: `internal/keychain/keychain_linux.go`, `internal/keychain/keychain_common.go`,
`internal/daemon/quota_rotation_dog.go`
**Problem**: `gt quota rotate` was macOS-only (Keychain). Linux Claude Code
stores OAuth tokens in `.credentials.json`.
**Fix**: Full Linux implementation reading/writing `.credentials.json` with
atomic writes (0600 perms). Added daemon patrol for automated rotation.
**Side effects**: Writes to `~/.claude/.credentials.json`. ExpiresAt field is
in milliseconds (not seconds) — validated. Daemon patrol interval configurable
via `daemon.json`.
**Upstream candidate**: Yes — Linux support is a gap.

## 6. Quota CLI help updates
**Commit**: `fb3c0109`
**Files**: `cmd/quota.go`
**Side effects**: None — docs only.

## 7. Dog execution model docs
**Commit**: `c5fd2fae`
**Files**: `docs/design/dog-execution-model.md`
**Side effects**: None — docs only.

## 8. Linear integration in mayor template
**Commit**: `77463a94`
**Files**: `internal/roles/templates/mayor.md.tmpl`
**Problem**: Mayor had no guidance on Linear issue sync.
**Fix**: Added `## Linear Integration` section to mayor role template.
**Side effects**: All mayors see Linear instructions. Harmless if Linear
MCP not configured.
**Upstream candidate**: Maybe — assumes Linear usage.

## 9. Daemon session ownership guard (GT_DAEMON_PID)
**Commit**: `93f958dc`
**Files**: `internal/daemon/daemon.go`, `internal/session/session.go`,
`internal/daemon/polecat.go`, `internal/daemon/crew.go`,
`internal/daemon/deacon.go`
**Problem**: Stale daemons killed sessions owned by newer daemons. Flock
prevents concurrent starts but can't stop an already-running old daemon.
**Fix**: Stamp `GT_DAEMON_PID` env var on all sessions. `KillExistingSession`
checks ownership — refuses to kill sessions owned by a different alive daemon.
On startup, daemon adopts orphaned sessions (owner PID dead).
**Side effects**: Sessions created outside daemon (manual `tmux new`) lack the
env var and will be adopted by the next daemon start. This is intentional.
**Upstream candidate**: Yes — real race condition fix.

## 10. tmux status bar length increase
**Commit**: `431818d8`
**Files**: `internal/session/tmux.go`
**Problem**: Emoji byte widths differ from display widths, causing tmux to
miscalculate truncation. With 4+ rigs, status-right exceeds 80 chars.
**Fix**: Bump `status-right-length` to 120, `status-left-length` to 30.
**Side effects**: Wider status bar. No functional impact.
**Upstream candidate**: Yes — simple improvement.

## 11. ASCII markers in status-line
**Commit**: `e4908546`
**Files**: `cmd/statusline.go`
**Problem**: tmux <=3.3 miscounts emoji display widths, causing status bar
to wrap and render multiple times.
**Fix**: Replace emoji with ASCII in status-line output (W=witness, R=refinery,
D=deacon, +=running, P=parked, X=docked).
**Side effects**: Status bar is less pretty. Rest of CLI keeps emoji.
**Upstream candidate**: Maybe — could be a config toggle.

## 12. Reset fds to blocking mode before syscall.Exec
**Commit**: `fc20fa5d`
**Files**: `internal/session/attach.go`, `cmd/agent.go`, `cmd/runtime.go`
**Problem**: Go sets stdin/stdout/stderr to non-blocking for its poller.
`syscall.Exec` inherits these. tmux 3.6+ rejects non-blocking fds with
"open terminal failed: not a terminal".
**Fix**: `unix.SetNonblock(fd, false)` on fds 0-2 before every `syscall.Exec`.
**Side effects**: None for normal operation. If Go's runtime poller was relying
on non-blocking fds at exec time, it's moot since exec replaces the process.
**Upstream candidate**: Yes — real compat fix for tmux 3.6+.

## 13. Upstream merge v0.11.0
**Commit**: `fffae54c`
**Conflict resolutions**:
- `daemon.go`: Dropped custom `restartPolecatSession` (upstream uses witness
  notification model).
- `polecat.md.tmpl`: Kept both RefPool section and upstream valid-statuses note.

## 14. `--to` flag for `gt quota rotate`
**Commit**: `cb3bfd84`
**Files**: `cmd/quota_rotate.go`, `internal/keychain/rotate.go`
**Problem**: LRU round-robin doesn't help when you know which account to target.
**Fix**: `--to <account>` directs rotation to a specific account.
**Side effects**: None — additive flag.
**Upstream candidate**: Yes.

## 15. Shared default tmux socket for cross-town navigation
**Commit**: `86fb1cf0`
**Files**: `internal/session/tmux.go`
**Problem**: v0.11.0 introduced per-town tmux sockets, which broke native
`Ctrl-b s` session switching across towns. Town-name-prefixed session names
(#2) already prevent collisions, making socket isolation redundant.
**Fix**: Revert to shared `"default"` socket when `GT_TMUX_SOCKET` unset.
`"auto"` still available for explicit isolation.
**Side effects**: All towns share one tmux server again. This is intentional —
session name prefixing handles isolation.
**Upstream candidate**: Yes — the upstream per-town socket design conflicts
with cross-town navigation.

## 16. Translate GT_DOLT_PORT → BEADS_DOLT_PORT in daemon
**Commit**: `c1a0f64f`
**Files**: `internal/daemon/daemon.go`, `internal/convoy/operations.go`,
`internal/daemon/dog_molecule.go`
**Problem**: Daemon sets `GT_DOLT_PORT` from `daemon.json`, but the beads SDK
(in-process) and `bd` subprocesses only read `BEADS_DOLT_PORT` /
`BEADS_DOLT_SERVER_PORT`. Multi-town setups using non-default ports (e.g., 3309)
hit port 3307, triggering circuit-breaker storms (11K+ failures in paper-town),
convoy scanner hot-looping, daemon recovery mode, and mayor session crash-loops.
**Fix**: Three locations:
1. `daemon.go`: After env propagation, translate `GT_DOLT_PORT` →
   `BEADS_DOLT_PORT` + `BEADS_DOLT_SERVER_PORT` in daemon process env.
2. `operations.go`: Pass `os.Environ()` to `bd` subprocess in
   `fetchCrossRigBeadStatus()`.
3. `dog_molecule.go`: Pass `os.Environ()` to `bd` subprocess in `runBd()`.
**Side effects**: None for single-town (default port 3307). Multi-town setups
now correctly route all Dolt traffic through the configured port.
**Upstream candidate**: Yes — critical bug fix for any non-default port config.
**Incident**: Paper-town mayor crash loop, 2026-03-08. Circuit breaker tripped
11K+ times, daemon entered recovery mode, mayor sessions reset every 30-60s
with increasing frequency. Root cause: convoy scanner hitting wrong port every
5s kept circuit breaker permanently open.

---

## Upgrade Procedure

When merging new upstream releases:

```bash
cd /data/gt/mayor/rig/gastown
git fetch origin
git log origin/main --oneline -10        # Review incoming changes
git merge origin/main -m "Merge origin/main (vX.Y.Z) into local/custom"
# Resolve conflicts — check this doc for affected files
make build
cp gt ~/.local/bin/gt
gt daemon stop && gt daemon start
```

**High-risk conflict areas**: `daemon.go` (session ownership + dolt port
translation), `session/registry.go` (town prefix), `tmux.go` (socket + status
bar), `statusline.go` (ASCII markers).
