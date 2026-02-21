# ruckus

`ruckus` is a safe-by-default Chaos Engineering CLI for **local Docker only**.

It is intentionally conservative:
- Dry-run planning is explicit via `plan`.
- Real execution requires `run ... --apply --yes-i-understand`.
- Targets must be allowlisted with `ruckus.enabled=true`.
- Every experiment is time-bounded.
- Revert steps always run at completion and on `Ctrl+C`.

## Scope (v1)

- Supported runtime: local Docker engine on the current machine.
- Not supported: remote SSH Docker hosts, Kubernetes.
- Persistence: SQLite at `~/.ruckus/ruckus.db`.

## Safety Model

- Default `--duration` is `30s`.
- Default `--interval` is `10s`.
- Hard duration cap is `5m`.
- To exceed `5m`, pass `--unsafe-max-duration`.
- Destructive execution gates:
  - `--apply`
  - `--yes-i-understand`
- Allowlist requirement:
  - Container must have label `ruckus.enabled=true`.
- Structured logs:
  - JSON logs by default to stdout.
  - `--human` toggles human-readable output for command responses.

## Install

Prerequisites:
- Go 1.24+
- Docker CLI + running local Docker daemon

Build:

```bash
go build -o ruckus ./cmd/ruckus
```

Run directly:

```bash
go run ./cmd/ruckus --help
```

## Experiments (v1)

### `kill-container`
- Behavior: repeatedly runs `docker restart` on the target at `--interval` for `--duration`.
- Revert: ensures a previously-running container is started again if needed.

### `net-latency`
- Behavior: applies `tc netem` delay/jitter in the target container namespace.
- Revert: removes the applied qdisc at completion/stop/cancel.
- Requirement: `tc` must exist in the target container.
- If `tc` is unavailable, this experiment returns a clear unsupported error.

### `cpu-stress`
- Behavior: starts a stress sidecar (`progrium/stress` by default) using target network namespace mode.
- Revert: force-removes the stress container.
- Fallback: host-level stress fallback is disabled by default.
- To enable host-level fallback, pass `--allow-host-stress`.

## Commands

### `ruckus targets`
Lists allowlisted containers (`ruckus.enabled=true`).

### `ruckus plan <experiment> [flags]`
Shows exactly what would happen; no changes are made.

### `ruckus run <experiment> --apply --yes-i-understand [flags]`
Executes experiment with safety acknowledgements.

### `ruckus stop <run-id>`
Requests stop and triggers revert for active run.

### `ruckus status`
Shows active and previous runs from local history.

## Examples

```bash
ruckus targets
```

```bash
ruckus plan kill-container --target myapp --duration 30s
```

```bash
ruckus run kill-container --target myapp --duration 30s --apply --yes-i-understand
```

```bash
ruckus status
```

```bash
ruckus stop <run-id>
```

## Label Allowlist

Only labeled containers can be targeted.

Example:

```bash
docker run -d --label ruckus.enabled=true --name myapp nginx:alpine
```

## Logging and History

- Actions are logged with:
  - `run_id`
  - timestamp
  - target
  - experiment
  - action/result
- Run history and events are stored in `~/.ruckus/ruckus.db`.

## Limitations

- v1 does not support Kubernetes.
- v1 does not support remote Docker hosts.
- `net-latency` depends on `tc` availability and permissions inside target.
- Host-level stress fallback can be dangerous and is disabled by default.

## Ambiguous/Safe Defaults Chosen

- Target selectors accept container name or ID accepted by `docker inspect`.
- `net-latency` defaults:
  - `--iface eth0`
  - `--latency 100ms`
  - `--jitter 20ms`
- `cpu-stress` defaults:
  - `--cpu-workers 1`
  - `--stress-image progrium/stress`
- Remote engine guard:
  - `DOCKER_HOST` must be unset or local (`unix://` / `npipe://`).
  - Any non-local `DOCKER_HOST` is rejected.

