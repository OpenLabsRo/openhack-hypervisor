# OpenHack Hypervisor

The control plane for OpenHack deployments. It automates the full lifecycle of
[`openhack-backend`](https://github.com/openlabsro/openhack-backend) instances —
syncing releases, staging configuration, running tests, building binaries,
provisioning systemd services, and reverse-proxying live traffic to the right
version — all driven by a REST API (and a companion host CLI).

It is built on [Fiber v3](https://github.com/gofiber/fiber), backed by MongoDB
(state) and Redis (cache), and integrates with **systemd** to run services and
**nginx** to terminate public traffic.

## Two binaries

- **`cmd/server`** — the hypervisor daemon. Serves the control API under
  `/hypervisor`, reverse-proxies everything else to managed backend
  deployments, and runs the stage/deployment lifecycle. Entrypoint flags mirror
  the backend: `--deployment`, `--port`, `--env-root`, `--app-version`.
- **`cmd/hyperctl`** — a host CLI for bootstrapping and operating the hypervisor
  itself (install, update, nginx/TLS, teardown). It must be run as root
  (`sudo`); see [hyperctl](#hyperctl).

## Core concepts

| Concept        | Meaning |
|----------------|---------|
| **Release**    | An immutable backend build target: a git tag + commit SHA. Releases are discovered by syncing the backend repo's tags. |
| **Stage**      | A configuration workspace identified by `<releaseID>-<envTag>` (e.g. `v25.10.27.0-dev`). It owns a dedicated repo checkout, an `.env` file on disk, and its test history. |
| **Test**       | A manual test run of a stage's checkout (the backend's `./TEST`), streamed live over WebSocket and recorded with a per-stage sequence number. |
| **Deployment** | A running instance of a stage (same id as the stage). It gets an allocated port, a compiled binary, and a systemd unit, and becomes routable. |
| **Event**      | A structured audit entry written to MongoDB on every operation. |

## Lifecycle

The end-to-end flow, all via the `/hypervisor` API (see `internal/core`):

1. **Sync releases** (`POST /hypervisor/releases/sync`) — runs
   `git ls-remote --tags` against the backend repo and records any new tags as
   `releases`.
2. **Create a stage** (`POST /hypervisor/stages`, `{releaseID, envTag}`) —
   clones the backend at the release's SHA into `/var/openhack/repos/<stageId>`,
   runs the backend's `./API_SPEC.sh`, seeds the stage `.env` from the template,
   and records the stage in status `pre`.
3. **Set the environment** (`PUT /hypervisor/stages/:stageId/env`) — writes the
   stage's `.env` to disk and moves it to status `ready`.
4. **Test** (`POST /hypervisor/stages/:stageId/tests`) — runs the backend's
   `./TEST.sh` against the stage checkout, streaming output over
   `GET /hypervisor/ws/stages/:stageId/tests/:sequence`. Tests are explicit;
   editing the env never auto-runs them.
5. **Deploy** (`POST /hypervisor/deployments/:stageId`) — allocates a port, runs
   the backend's `./BUILD.sh` into `/var/openhack/builds/<version>`, installs and
   starts a systemd unit, and (asynchronously) marks the deployment `ready`. A
   ready deployment is immediately reachable at `/<stageId>/*`.
6. **Promote** (`POST /hypervisor/deployments/:deploymentId/promote`) — makes a
   deployment the **main** one, so the root path `/` proxies to it. Promotion is
   always an explicit operator action.

Deployments can also be stopped, started, and deleted; stages can be deleted
(which tears down their checkout, env, and test logs).

## Routing

The daemon is itself the reverse proxy (`internal/proxy`):

| URL prefix      | Target |
|-----------------|--------|
| `/hypervisor/*` | The control API and WebSocket streams (handled in-process) |
| `/<stageId>/*`  | The matching ready deployment, on its `localhost:<port>` |
| `/`             | The **main** (promoted) deployment |

The route map is rebuilt from the `deployments` collection on startup and
updated as deployments change. This version-prefixed routing is what the
backend's Swagger version-stamping aligns with (`NO_HYPER`).

`/hypervisor/meta/drain` toggles **drain mode** for blue-green cutovers: while
draining, `meta/ping` returns `503` so an upstream load balancer stops sending
new traffic.

## Filesystem layout

Managed under two roots (`internal/paths`):

```
/var/hypervisor/        # the hypervisor's own assets
  repos/  builds/  env/  logs/
/var/openhack/          # backend assets, managed per stage/deployment
  repos/<stageId>/      # one checkout per stage
  builds/<version>      # compiled backend binaries
  env/template/.env     # canonical env template
  env/<stageId>/.env    # per-stage environment
  runtime/logs/         # test + deployment log files
```

systemd units are written to `/lib/systemd/system`.

## Data stores

- **MongoDB** database `hypervisor` (`hypervisor_dev` for the `dev` profile,
  `hypervisor_tests` for `test`). Collections: `hyperusers`, `git_commits`,
  `releases`, `stages`, `tests`, `deployments`, `events`.
- **Redis** at `127.0.0.1:6379`, logical DB `15`.

## Configuration

Configuration comes from a `.env` file (loaded via `godotenv`) plus a `VERSION`
file at the repo root. Keys consumed by the daemon:

| Key                     | Purpose |
|-------------------------|---------|
| `MONGO_URI`             | MongoDB connection string |
| `JWT_SECRET`            | Secret used to verify hyperuser tokens |
| `GITHUB_WEBHOOK_SECRET` | Secret for GitHub webhook verification |
| `PREFORK`               | Enables Fiber prefork mode when `true` |
| `REPO_URL`              | Backend repo to clone/sync (defaults to `https://github.com/OpenLabsRo/openhack-backend`) |

The listen **port** and **deployment profile** are passed as CLI flags, not env
vars.

## Running locally

Requires Go (see `go.mod`), plus reachable MongoDB and Redis. Note that the full
stage/deployment lifecycle drives `git`, `systemd`, and the filesystem under
`/var`, so it is meant to run on a managed host; locally you can run the API and
exercise the read/sync paths.

```bash
# dev profile on port 8080, using the repo .env (see RUNDEV.sh)
./RUNDEV.sh

# or directly
go run ./cmd/server --deployment dev --port 8080
```

## Testing

Integration specs live under `test/`, run against the `test` profile
(`hypervisor_tests` DB). MongoDB and Redis must be running, seeded with the
`testhyperuser` account.

```bash
./TEST.sh                    # go test ./test/... -v -count=1
```

`DEP_WS.sh` and `TEST_WS.sh` are convenience scripts that open a `wscat`
connection to the deployment-log and test-log WebSocket streams.

## Build & Release

| Script             | Purpose |
|--------------------|---------|
| `./BUILD.sh`       | Builds a stripped, static `linux/amd64` daemon binary into `bin/<VERSION>` |
| `./TEST.sh`        | Runs the Go test suite |
| `./API_SPEC.sh`    | Regenerates the Swagger docs via `swag init` |
| `./RUNHYPERCTL.sh` | Builds and runs the `hyperctl` CLI locally |

Versions follow a `YY.MM.DD.B` scheme. To cut a release, a developer runs
`./RELEASE.sh`: it bumps `VERSION`, builds the `hyperctl` binary, commits, tags
`v<version>`, pushes, and publishes a GitHub release with the `hyperctl` binary
attached as an asset.

## hyperctl

`hyperctl` manages the hypervisor on a host machine and must be run with `sudo`.
Commands:

| Command      | Purpose |
|--------------|---------|
| `manhattan`  | Bootstrap or update the hypervisor service |
| `trinity`    | Update the hypervisor by cloning, testing, and building a new version |
| `grimhilde`  | Update `hyperctl` itself to the latest version |
| `swaddle`    | Generate the nginx configuration for the hypervisor |
| `knox`       | Secure nginx with an SSL certificate via certbot |
| `interstate` | Show the current routing map |
| `ping`       | Ping the hypervisor health endpoint |
| `nagasaki`   | Stop the running hypervisor service |
| `hiroshima`  | Remove all hypervisor and OpenHack directories (destructive) |
| `version`    | Show the installed hypervisor build |
| `help`       | Show usage |

## API documentation

Swagger UI is served at `/hypervisor/docs`, with the spec at
`/hypervisor/docs/doc.json` (base path `/hypervisor`). All state-changing routes
require a hyperuser JWT (`Authorization: Bearer <token>`); WebSocket streams
authenticate via an `authorization` query parameter.
</content>
