# Hypervisor Deployment Plan

## Goals

- Proxies public traffic to the latest healthy OpenHack backend using blue/green slots.
- Automates build, test, deploy, and traffic switching using GitHub webhooks + manual promotion.
- Enforces hyperuser authentication (mirrors backend superuser flow) without storing long-lived tokens.
- Survives host reboots via systemd; manages backend instances without manual sudo interaction.

## Service Layout

- **Hypervisor daemon (Fiber)**

  - Runs under a system-level systemd unit (`openhack-hypervisor.service`).
  - Terminates TLS/HTTP, forwards requests to the active backend color.
  - Stores state in MongoDB `hypervisor` database: release metadata, deployment records, GitHub events, hyperuser sessions.
  - Exposes APIs for webhook ingestion, deployment control, status/health, and hyperuser login.
  - On startup reconciles DB state, restarts the previously active color, verifies health, and resumes proxying.

- **Backend colors (blue/green)**

  - Managed by the hypervisor via user-level systemd units (`openhack-backend-blue.service`, `...-green.service`).
  - Each unit references a release-specific working directory and artifact built by the hypervisor.
  - Ports stay stable per color (e.g., blue `8001`, green `8002`). Hypervisor flips traffic by switching the active color in its reverse proxy layer.
  - Health checks: readiness probe (`/ping`/`/version`), optional smoke tests before promotion, continuous liveness polling for automatic restart/redploy.

- **Command-line tools**
  - `hyperctl`: server-resident utility. `setup` clones/builds/deploys the daemon, writes/upgrades systemd units, seeds config, enables lingering; `upgrade` pulls and rebuilds the hypervisor before restarting it; `restart`/`status` (and optional `logs`, `reconcile`) wrap systemd and local health checks. Runs locally and requires no auth.
  - `hyper-cli`: operator CLI used from remote machines. Authenticates as hyperuser for every session, lists release tags, inspects build/test logs, stages builds, and promotes or rolls back traffic via `switch`.
  - Future additions: `hyper-cli logs`, `hyper-cli rollback`, `hyper-cli config` for expanded remote control.

## Release & Deployment Flow

1. **Git tag**: Developers run the new `RELEASE` script locally.

   - Auto-bumps `VERSION` (format `YY.MM.DD.B`, zero-indexed per day) and commits/tags the change (tag `v<version>`).
   - Script supports manual version overrides and `--no-git` dry runs.

2. **GitHub webhook**: Hypervisor listens for push/tag webhook events.

   - Validates signature, records tag metadata (name, SHA, author, timestamp) in Mongo.
   - Marks tag as deployable candidate when it matches release template (tag + `VERSION` change).

3. **Operator selects tag** using `hyper-cli` or the dashboard.

   - Hypervisor clones repo into `/var/openhack/repos/<tag>` and checks out target SHA.
   - Runs repo’s `BUILD` script to produce `/var/openhack/artifacts/<tag>/<tag>` binary.
   - Executes repo’s `TEST` script with flags `--env-root`/`--app-version`; stores logs and exit status.
   - On success, updates inactive color’s systemd unit to point at new artifact and env root, restarts service, confirms health checks pass.

4. **Traffic switch**: Once the new color is marked healthy, operator issues `hyper-cli switch`.

   - Hypervisor updates proxy routing to the new color, drains old color, and keeps it warm for instant rollback.

5. **Rollback**: `hyper-cli switch --to <previous>` reassigns traffic to prior color if needed. Hypervisor retains history so previous version can be reactivated rapidly.

## Environment & Configuration

- Hypervisor-managed systemd units pass flags instead of environment files:
  - `ExecStart=/var/openhack/artifacts/<tag>/<tag> --deployment prod --port 8001 --env-root /var/openhack/env --app-version <tag>`.
- Shared `.env` lives at `/var/openhack/env/.env`. Hypervisor (and RUNDEV/TEST scripts) load this by passing `--env-root` flag.
- Per-release adjustments (if needed) can be encoded in version-specific env directories, but default flow uses one global env root.

## Backend Adjustments (openhack-backend)

- **Environment Loader**

  - `internal/env/env.go` now accepts flags (`--env-root`, `--app-version`) and defaults to repo root for dev/test runs. Superuser defaults remain baked in for tests.
  - `env.Init` loads `.env` from the provided root and version from either flag or `VERSION` file.

- **Entry Point**

  - `cmd/server/main.go` uses Go flags for deployment, port, env root, and app version. Port must be supplied by caller (no implicit defaults).
  - `/version` endpoint reports the supplied version metadata.

- **Helper Scripts**

  - `RUNDEV`: launches `go run` with dev profile, port 9001, repo env root, and version from `VERSION`.
  - `TEST`: forwards optional `--env-root`/`--app-version` flags to the test suite; defaults point to repo assets. Resets Redis counter before tests.
  - `BUILD`: compiles the backend binary with release flags, producing `<project-root>/<version>` by default or `<output>/<version>` when `--output DIR` is provided.
  - `RELEASE`: bumps or sets `VERSION` (format `YY.MM.DD.B`), writes commit/tag/ push by default (opt-out via `--no-git`). Tag format `v<version>`.

- **Testing changes**

  - Tests now register the shared env flags, run under package-specific `TestMain` (e.g., `test/superusers/test_main.go`) to build an app instance.
  - Suites refactored to use shared setup helpers instead of pseudo `TestMain` functions.

- **Misc**
  - `api_spec.yaml` refreshed; `contract.yaml` replaced/renamed.
  - `.air.toml` updated to run `./RUNDEV` directly instead of rebuilding.
  - `.gitignore` adjusted (removed `.env` ignore to keep repo-local env?).

## Systemd Strategy

- Hypervisor service (`/etc/systemd/system/openhack-hypervisor.service`) handles restarts; `hyperctl setup` installs this unit with `ExecStart=/usr/local/bin/openhack-hypervisor --config /etc/openhack/hypervisor.yaml` and enables it on boot.
- Blue/green units run as the deployment user via `systemctl --user`. Hypervisor (and `hyperctl setup/upgrade`) write unit files under `~/.config/systemd/user/`, run `systemctl --user daemon-reload`, and `start`/`stop`/`restart` them as deployments progress.
- Ports: allocate fixed per color (e.g., blue 8001, green 8002, optional canary 8003). Hypervisor stores mapping in Mongo/config.

## Data & State

- MongoDB `hypervisor` DB collections:
  - `releases`: tag metadata, build/test results, artifact paths, env settings, timestamps.
  - `deployments`: current and historical blue/green assignments, active color pointer, health statuses.
  - `hyperusers`: superuser credential mirror (hashed passwords, roles).
  - `sessions`: issued hyperuser tokens with TTL.
  - `github_events`: raw webhook payloads/IDs to avoid replays.

## Future Considerations

- Add smoke test script integration to hypervisor pipeline (config-driven command per tag).
- Track build/test logs in object storage (S3/minio) for long-term auditing.
- Extend `hyperctl`/`hyper-cli` with log tailing, config editing, and GitHub event inspection.
- Implement notifications for failed builds/tests (Slack/email).

This document aggregates our current plan and repository adjustments so future work on the hypervisor stack has a single reference point.
