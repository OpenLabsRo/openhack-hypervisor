# ⚙️ OpenHack Hypervisor — System Design & Specifications# ⚙️ OpenHack Hypervisor — Final Contract Specification

**Version:** 2025-10-19 **Version:** 2025-10-17

**Purpose:** automate build, test, deploy, routing, and lifecycle management of the Hypervisor daemon and OpenHack backend services via a unified REST API and CLI tooling. **Purpose:** automate build, test, deploy, routing, and lifecycle management of OpenHack backend services with full control from a web interface.

**Tech stack:** Go (Fiber + MongoDB + Redis) + systemd. **Tech stack:** Go (Fiber + MongoDB) + systemd.

**API base URL:** `https://DOMAIN/hypervisor/*`**Public base URL:** `https://DOMAIN/hypervisor/*`

---

## 1. Core Concepts## 1. Core concepts

- **Release** — immutable build artifact of a tagged commit (Hypervisor or OpenHack backend).- **Release** — immutable build artifact of a tagged commit.

- **Stage** — configuration workspace (`version + envTag`) that owns the environment file, repo clone, test history, and lifecycle status (`pre`, `active`, `promoted`).- **Stage** — configuration workspace (`version + envTag`) that owns the environment file, repo clone, test history, and lifecycle status (`pre`, `active`, `promoted`).

- **Stage test result** — outcome of a manual test run (started via API, streamed over WebSocket) linked to the stage that triggered it.- **Stage test result** — outcome of a manual test run (started via API, streamed over WebSocket) linked to the stage that triggered it.

- **Deployment** — running instance of a stage, uniquely identified by `vVERSION-ENVTAG` (e.g. `v25.10.17.0-prod`); deployments are created only from `active` stages and managed via systemd.- **Deployment** — running instance of a stage, uniquely identified by `vVERSION-ENVTAG` (e.g. `v25.10.17.0-prod`); deployments are created only from `active` stages.

- **Event** — structured log entry emitted on every operation (for auditing and correlation).- **Event** — structured log entry emitted on every operation (for auditing and correlation).

- **Hypervisor API** — all control routes served under `/hypervisor`.- **Routing** — URL prefixes mapped to deployments; `/` points to the promoted “main” one.

- **Hyperctl** — CLI tool for bootstrapping, managing, and decommissioning the Hypervisor system on a host machine.- **Hypervisor API** — all control routes served under `/hypervisor`.

---

## 2. Filesystem Layout## 2. Filesystem layout

### Hypervisor Directories```

/var/openhack/

```repos/

/var/hypervisor/    v25.10.17.0/               # cloned at tag

  repos/  builds/

    main/                      # cloned hypervisor repository    v25.10.17.0                # compiled binary

  builds/  env/

    v25.10.19.0                # compiled hypervisor binary    template/.env              # canonical staging template

  env/    v25.10.17.0-dev/.env

    .env                       # hypervisor runtime environment    v25.10.17.0-prod/.env

  logs/  runtime/

    hyperctl.log    logs/

    hypervisor.log    tmp/

```

/etc/openhack/versions/

### OpenHack Backend Directories v25.10.17.0-dev/env # BIN/DEPLOYMENT/PORT/ENV_ROOT/APP_VERSION + user env

```

```

/var/openhack/Per-deployment environment files live under `/var/openhack/env`. There is a single canonical template at `/var/openhack/env/template/.env`. Each deployment uses exactly one environment file derived from the template; the file is located under `/var/openhack/env/<deployment-id>/.env`.

repos/

    v25.10.17.0/               # cloned backend at tag---

builds/

    v25.10.17.0                # compiled backend binary## 3. Build & test conventions

env/

    template/.env              # canonical staging template### BUILD

    v25.10.17.0-dev/.env       # per-stage environment

    v25.10.17.0-prod/.env```

```./BUILD [--output DIR]

# Example:

Per-deployment environment files live under `/var/openhack/env`. There is a single canonical template at `/var/openhack/env/template/.env`. Each stage deployment uses exactly one environment file derived from the template; the file is located under `/var/openhack/env/<stageId>/.env`../BUILD --output /var/openhack/builds

```

---

- Reads local `VERSION` file.

## 3. Hyperctl — CLI Tooling Specifications- Builds executable at `${OUTPUT}/${VERSION}` (e.g. `/var/openhack/builds/v25.10.17.0`).

- Exit `0` on success.

**Hyperctl** is a standalone CLI companion to the Hypervisor API, designed to be run manually or via automation on a host machine to bootstrap and manage the entire system lifecycle.

### TEST

### 3.1 Command Overview

````

```./TEST [--env-root PATH] [--app-version VERSION]

Usage: hyperctl <command> [options]# Example:

./TEST --env-root /var/openhack/env/v25.10.17.0-dev --app-version v25.10.17.0-dev

Available commands:```

  setup      Bootstrap or update the hypervisor service

  nagasaki   Stop the running hypervisor service (graceful shutdown)- Reads env from `${PATH}/.env`.

  hiroshima  Completely remove all hypervisor and OpenHack directories (destructive)- Prints logs to stdout/stderr.

  version    Show the currently installed hypervisor build- Exit `0` on success.

  help       Show this help text- Tests run only when explicitly started through the stage testing API; creating or updating a stage does not auto-run tests.

````

---

### 3.2 Setup Command

## 4. Executable runtime flags

**Purpose:** Bootstrap the Hypervisor system from scratch or update an existing installation.

````

```bash--deployment <envTag>

hyperctl setup [--dev]--port <port>

```--env-root </var/openhack/env/vTAG-ENV>

--app-version <vTAG-ENV>

**Flags:**```

- `--dev` (optional): Development mode; uses current directory instead of cloning the hypervisor repository. Skips test execution.

---

**Workflow:**

## 5. systemd integration

1. **Prerequisite check** — Verify that required system tools are available (git, go, systemctl, sudo, etc.).

2. **Directory creation** — Create `/var/hypervisor` and `/var/openhack` directory trees with appropriate permissions.Unit: `openhack-backend@.service` (installed into the systemd unit directory defined in the codebase via `paths.SystemdUnitDir`)

3. **Environment configuration** — Interactively prompt the operator to create or edit `/var/hypervisor/env/.env` if missing (using `$EDITOR`).

4. **Repository cloning** — Clone the hypervisor repository from `https://github.com/OpenLabsRo/openhack-hypervisor` into `/var/hypervisor/repos/main`. In dev mode, use the current working directory.```

5. **Test execution** — Run the test suite (`go test ./...`) to verify the codebase is functional. Skipped in dev mode.[Unit]

6. **Build** — Execute the repository's `BUILD` script to compile the hypervisor binary and output it to `/var/hypervisor/builds/`.Description=OpenHack backend %i

7. **Systemd unit installation** — Generate and install the hypervisor systemd unit file (`openhack-hypervisor.service`) with configuration:After=network-online.target

   - Binary path from build outputWants=network-online.target

   - Deployment tag (default: `prod`)

   - Port (default: `8080`)[Service]

   - Environment root (`/var/hypervisor/env`)Type=exec

   - Application version from build output

8. **State persistence** — Save the installation state (version, binary path) to `/var/hypervisor/state.json` for reference by other commands.# The environment file for an individual deployment lives at /var/openhack/env/<deployment-id>/.env

9. **Systemd reload** — Execute `systemctl daemon-reload` and `systemctl enable openhack-hypervisor.service`.EnvironmentFile=/var/openhack/env/%i/.env

10. **Service startup** — Start or restart the hypervisor service via `systemctl restart openhack-hypervisor.service`.ExecStart=${BIN} \

11. **Health verification** — Poll the service's health endpoint (`GET /hypervisor/meta/ping`) to confirm the daemon is operational. --deployment ${DEPLOYMENT} \

12. **Completion** — Print success message with next steps. --port ${PORT} \

 --env-root ${ENV_ROOT} \

**Example output:** --app-version ${APP_VERSION}

Restart=always

```RestartSec=1

Starting hypervisor setup...DynamicUser=yes

Checking system prerequisites...NoNewPrivileges=yes

Ensuring directory layout...ProtectSystem=strict

Directory layout readyProtectHome=read-only

[Editor opens for /var/hypervisor/env/.env]PrivateTmp=yes

Cloning the project...StateDirectory=openhack-%i

Project cloned to /var/hypervisor/repos/mainRuntimeDirectory=openhack-%i

Testing the code...

[Tests run...][Install]

Building the code into /var/hypervisor/builds...WantedBy=multi-user.target

Build complete: /var/hypervisor/builds/v25.10.19.0```

Installing systemd unit for hypervisor...

Systemd unit installedEnvironment example:

Persisting installation state...

State saved```

Reloading systemd...BIN=/var/openhack/builds/v25.10.17.0

Systemd reloadedDEPLOYMENT=dev

Checking service status...PORT=20037

Service is activeENV_ROOT=/var/openhack/env/v25.10.17.0-dev

Performing health check...APP_VERSION=v25.10.17.0-dev

Health check passed# --- begin user env ---

Setup completed successfully.PREFORK=false

```MONGO_URI=mongodb+srv://...

JWT_SECRET=abcd

### 3.3 Nagasaki CommandBADGE_PILES=6

# --- end user env ---

**Purpose:** Gracefully stop the running Hypervisor service without removing any data.```



```bash---

hyperctl nagasaki

```## 6. MongoDB collections



**Flags:** None.### releases



**Workflow:**```

{

1. Execute `systemctl stop openhack-hypervisor.service`.  "id": "v25.10.17.0",

2. Verify the service has stopped.  "sha": "abc123",

3. Print success message.  "artifact": { "binPath": "/var/openhack/builds/v25.10.17.0" },

  "createdAt": "2025-10-17T13:00:00Z"

**Example output:**}

````

```

Stopping hypervisor service...### stages

Hypervisor service stopped.

```

{

### 3.4 Hiroshima Command "id": "v25.10.17.0-dev",

"releaseId": "v25.10.17.0",

**Purpose:** Completely remove the Hypervisor installation (destructive operation). "envTag": "dev",

"status": "pre|active|promoted",

````bash

hyperctl hiroshima [--yes]  "createdAt": "...",

```  "updatedAt": "...",

  "lastTestResultId": "tr_7yQ"    # nullable; most recent manual test run

**Flags:**}

- `--yes` (optional): Skip confirmation prompt and proceed with destruction.```



**Workflow:**Environment contents are stored on disk at `/var/openhack/env/<stageId>/.env`; the API reads and writes this file directly.



1. **Safety confirmation** — Unless `--yes` is provided, display a warning and prompt for explicit user confirmation ("Type 'yes' to confirm").### stage_test_results

2. **Stop service** — Execute `systemctl stop openhack-hypervisor.service` (continue on error).

3. **Disable service** — Execute `systemctl disable openhack-hypervisor.service` (continue on error).```

4. **Remove systemd unit** — Delete `/lib/systemd/system/openhack-hypervisor.service`.{

5. **Reload systemd** — Execute `systemctl daemon-reload`.  "id": "tr_7yQ",

6. **Remove `/var/hypervisor`** — Recursively delete the directory and all contents.  "stageId": "v25.10.17.0-dev",

7. **Remove `/var/openhack`** — Recursively delete the directory and all contents.  "status": "running|passed|failed|canceled|error",

8. **Completion** — Print confirmation message.  "wsToken": "...",

  "logPath": "/var/openhack/runtime/logs/tr_7yQ.log",

**Example output:**  "startedAt": "...",

  "finishedAt": null

```}

WARNING: This command will completely remove /var/hypervisor and /var/openhack directories.```

This action is irreversible and will delete all data, configurations, and builds.

Are you sure? Type 'yes' to confirm: yes### deployments

Stopping hypervisor service...

Disabling hypervisor service...```

Removing systemd service file...{

Reloading systemd...  "id": "v25.10.17.0-dev",

Removing /var/hypervisor...  "stageId": "v25.10.17.0-dev",

Removing /var/openhack...  "version": "v25.10.17.0",

Uninstallation complete. All hypervisor and OpenHack data has been removed.  "envTag": "dev",

```  "port": 20037,                  # nullable; staged deployments have no port yet

  "status": "staged|ready|stopped|deleted",

### 3.5 Version Command  "createdAt": "...",

  "promotedAt": null

**Purpose:** Display the currently installed hypervisor build version.}

````

```bash

hyperctl version---

```

## 7. Event system

**Flags:** None.

All operations emit structured events for auditing and replay. Events are persisted to the MongoDB `events` collection by the asynchronous emitter in `internal/events`.

**Workflow:**

### Event schema

1. Read the saved state from `/var/hypervisor/state.json`.

2. Extract and print the version field.The canonical shape is `internal/models.Event`:

3. If state file is missing or unreadable, print an error.

````

**Example output:**type Event struct {

    TimeStamp  time.Time         `json:"timestamp" bson:"timestamp"`

```    Action     string            `json:"action" bson:"action"`

v25.10.19.0    ActorID    string            `json:"actorID" bson:"actorID"`

```    ActorRole  string            `json:"actorRole" bson:"actorRole"`   // e.g. "hyperuser", "system"

    TargetID   string            `json:"targetID" bson:"targetID"`

### 3.6 Help Command    TargetType string            `json:"targetType" bson:"targetType"` // e.g. "stage", "deployment"

    Props      map[string]any    `json:"props" bson:"props"`

**Purpose:** Display usage information.    Key        string            `json:"key,omitempty" bson:"key,omitempty"`

}

```bash```

hyperctl help

# orBefore persistence the emitter stamps every event with the `Europe/Bucharest` timezone so audit timelines align with operations.

hyperctl --help

hyperctl -h### Emitter behaviour

````

- `events.Em` is initialised during application bootstrap once Mongo connectivity is available.

Prints the command list and basic usage.- Events are buffered through an internal channel (default capacity 1000) and flushed in batches of 50 via `InsertMany`. A timer (2 s in normal deployments, 50 ms in the `test` profile) ensures periodic flushes even when the batch is not full.

- If the buffer is saturated the emitter falls back to a synchronous `InsertOne`.

---- On shutdown `Emitter.Close()` drains the buffer to avoid data loss.

- Convenience wrappers (`internal/events/*.go`) set consistent actor/target constants such as `ActorHyperUser`, `ActorSystem`, ensuring consumers see predictable `actorRole`/`targetType` values.

## 3.7 Hyperctl Directory Structure

Example event:

````

internal/hyperctl/```

├─ commands/{

│  ├─ setup.go               # RunSetup orchestration  "timestamp": "2025-10-17T12:34:56+02:00",

│  ├─ nagasaki.go            # RunNagasaki graceful shutdown  "action": "stage.env_updated",

│  ├─ hiroshima.go           # RunHiroshima destructive cleanup  "actorID": "user_123",

│  ├─ version.go             # RunVersion display  "actorRole": "hyperuser",

│  └─ usage.go               # PrintUsage help text  "targetID": "v25.10.17.0-dev",

├─ build/  "targetType": "stage",

│  └─ build.go               # Executes BUILD script, parses VERSION  "props": {}

├─ fs/}

│  └─ fs.go                  # Directory creation, env editing, file removal```

├─ git/

│  └─ git.go                 # Repository cloning and pullingTypical events include:

├─ health/

│  └─ health.go              # Polls /hypervisor/meta/ping endpoint- `stage.prepared`, `stage.env_updated`, `stage.failed`

├─ system/- `stage.test_started`, `stage.test_passed`, `stage.test_failed`, `stage.test_canceled`

│  └─ system.go              # Prerequisite checks, editor resolution- `deployment.created`, `deployment.create_failed`

├─ systemd/- `deploy.started|ready|health_failed|stopped|deleted`

│  ├─ manage.go              # Systemd unit templating and service control- `promote.changed`

│  ├─ files.go               # Embedded service file assets- `sync.started|release_created|failed|finished`

│  └─ openhack-hypervisor.service # Unit template

├─ state/All code paths must emit via `events.Em.Emit(...)` (or the provided wrappers) to benefit from batching and schema enforcement.

│  └─ state.go               # Persistence of installation state

└─ testing/---

   └─ testing.go             # Runs `go test ./...`

```## 8. API (all under `/hypervisor`)



### 3.8 Hyperctl Subsystem Specifications| Endpoint                                       | Description                                                                                                                                         |

| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |

#### build.go| `GET /hypervisor/meta/ping`                 | Hypervisor health probe (PONG)                                                                                                                      |

| `GET /hypervisor/meta/version`              | Running hypervisor version string                                                                                                                   |

- **Function:** `Run(repoDir, outputDir) (Result, error)`| `GET /hypervisor/hyperusers/whoami`          | Return the authenticated hyperuser profile (JWT required)                                                                                            |

- **Behavior:**| `GET /releases`                              | List all releases                                                                                                                                    |

  - Reads `VERSION` from the repo root.| `POST /stages`                                 | Create a **pre-stage**: clones the release repo, seeds the template env on disk, and returns stage metadata plus the seeded env                      |

  - Constructs output path: `${outputDir}/${VERSION}`.| `GET /stages`                                  | List stages with status and last test summary (env text retrieved separately)                                                                      |

  - Deletes any existing binary at that path.| `GET /stages/:stageId`                         | Inspect a single stage (current env text, status timeline, last test results)                                                                      |

  - Executes `bash BUILD --output ${outputDir}` from the repo directory.| `GET /stages/:stageId/env`                     | Read the current stage `.env` contents from disk                                                                                                   |

  - Captures stdout/stderr and forwards to the caller's stdout/stderr.| `PUT /stages/:stageId/env`                     | Replace the stage `.env` file; first update moves the stage from `pre` → `active`                                                                  |

  - Returns `Result{Version, BinaryPath}` on success or error on failure.| `POST /stages/:stageId/tests`                  | Start a manual test run using the current stage env; response returns `{ testResultId, wsToken }` for streaming                                    |

| `POST /stages/:stageId/tests/:resultId/cancel` | Cancel an in-flight stage test (optional, when supported)                                                                                           |

#### fs.go| `GET /.ws/stages/:stageId/tests/:resultId`     | WebSocket for live `./TEST` output associated with the stage test                                                                                   |

| `POST /deployments`                            | Promote an `active` stage to a deployment (writes build artifacts + systemd unit, records deployment referencing `stageId`)                         |

- **Functions:**| `GET /deployments`                             | List active deployments                                                                                                                             |

  - `EnsureLayout() error` — Creates `/var/hypervisor` and `/var/openhack` with full subdirectory trees.| `GET /deployments/main`                        | Get main deployment                                                                                                                                 |

  - `EnsureEnvDirFor(envPath) error` — Creates parent directories for an environment file if missing.| `POST /deployments/:deploymentId/promote`      | Promote an existing deployment to become the main route                                                                                             |

  - `EditEnvFileIfMissing(label, envPath, editor) error` — Opens the file in `$EDITOR` if it does not exist; creates with boilerplate if editor succeeds.| `POST /deployments/:deploymentId/shutdown`     | Stop process, keep record                                                                                                                           |

  - `HypervisorEnvPath() string` — Returns `/var/hypervisor/env/.env`.| `DELETE /deployments/:deploymentId`            | Delete deployment (`?force=true` stops first)                                                                                                       |

  - `RemoveDir(path) error` — Recursively removes a directory with error handling.| `GET /routes/main`                             | Get main route info                                                                                                                                 |

| `PUT /routes/main`                             | Set main deployment                                                                                                                                 |

#### git.go| `POST /sync`                                   | Bulk sync commits and releases                                                                                                                      |

| `GET /hypervisor/docs` / `GET /hypervisor/docs/doc.json` | Swagger UI + generated spec                                                                                                         |

- **Function:** `CloneOrPull(repoURL, targetDir) error`

- **Behavior:**- All write routes use JWT Bearer auth.

  - If `targetDir` exists and contains a `.git` folder, runs `git pull` in that directory.- WebSocket auth via `?token=` query param.

  - Otherwise, runs `git clone ${repoURL} ${targetDir}`.

  - Returns error on failure.---



#### health.go## 9. Routing behavior



- **Function:** `Check() error`| URL prefix            | Target                         | Purpose         |

- **Behavior:**| --------------------- | ------------------------------ | --------------- |

  - Polls `http://localhost:8080/hypervisor/meta/ping` (or port from state).| `/v25.10.17.0-dev/*`  | localhost:20037                | dev deployment  |

  - Retries up to 30 times with 1-second intervals.| `/v25.10.17.0-prod/*` | localhost:20055                | prod deployment |

  - Returns success if any attempt returns "PONG", error if all fail.| `/`                   | whichever deployment is “main” | public route    |



#### system.goMultiple deployments of the same version coexist.



- **Functions:**---

  - `CheckPrerequisites() error` — Verifies availability of: `git`, `go`, `systemctl`, `bash`, and `sudo`.

  - `ResolveEditor() string` — Returns the editor from `$EDITOR` environment variable, or defaults to `vi` if unset.## 10. Swagger documentation



#### systemd/manage.go- Uses swaggo tooling; generated via `./API_SPEC` which wraps `swag init`.

- Output lives in `internal/swagger/docs/`.

- **Type:** `ServiceConfig{BinaryPath, Deployment, Port, EnvRoot, Version}`- Served at `/hypervisor/docs` (HTML UI) and `/hypervisor/docs/doc.json` (raw spec).

- **Functions:**- `BasePath: /hypervisor`

  - `InstallHypervisorService(cfg ServiceConfig) error` — Writes the unit file, reloads systemd, enables and restarts the service.- Security: `BearerAuth` (Authorization header)

  - `writeHypervisorUnit(cfg ServiceConfig) error` — Parses the embedded unit template, renders it with the config values, and writes to `/lib/systemd/system/openhack-hypervisor.service`.

  - `StopHypervisorService() error` — Runs `systemctl stop openhack-hypervisor.service`.---

  - `DisableHypervisorService() error` — Runs `systemctl disable openhack-hypervisor.service`.

  - `RemoveServiceFile() error` — Deletes `/lib/systemd/system/openhack-hypervisor.service`.## 11. Project structure

  - `ReloadSystemd() error` — Runs `systemctl daemon-reload`.

  - `CheckServiceStatus() error` — Runs `systemctl is-active openhack-hypervisor.service` and returns error if not active.```

openhack-hypervisor/

#### state.go├─ cmd/server/

│  └─ main.go                 # fiber app, mount /hypervisor, swagger, routes

- **Type:** `State{Version, BuildPath}`├─ internal/

- **Functions:**│  ├─ api/                    # HTTP layer + swagger annotations (stages, deployments, tests, sync, hyperusers)

  - `Save(state State) error` — Marshals and writes the state struct to `/var/hypervisor/state.json`.│  ├─ core/                   # business logic (stage lifecycle, deployments, tests, routing, sync)

  - `Load() (State, error)` — Reads and unmarshals `/var/hypervisor/state.json`.│  ├─ db/                     # Mongo + Redis initialization

│  ├─ env/                    # runtime flags/env loading

#### testing.go│  ├─ events/                 # reusable event emitter

│  ├─ hyperusers/             # authentication handlers

- **Function:** `RunTests(repoDir) error`│  ├─ models/                 # Mongo persistence models (stage, test result, deployment, release, etc.)

- **Behavior:**│  ├─ swagger/                # embedded swagger assets + UI routes

  - Executes `go test ./...` from the repo directory.│  └─ utils/                  # helpers (errors, locals, ids)

  - Captures and forwards stdout/stderr.├─ API_SPEC                   # helper script to regenerate swagger docs

  - Returns error on test failure.├─ HYPERVISOR_DESIGN.md       # this contract

├─ go.mod / go.sum

---└─ VERSION

````

## 4. Hypervisor Systemd Unit Template

---

**File:** `internal/hyperctl/systemd/openhack-hypervisor.service`

## 12. Environment variables

````ini

[Unit]These are the minimal environment variables consumed by the hypervisor and the build/test/deploy tooling. Backend deployments will additionally read their per-deployment `.env` from `/var/openhack/env/<deployment-id>/.env`.

Description=OpenHack Hypervisor

After=network.target redis.service```

Requires=redis.serviceHTTP_ADDR=:9090

MONGO_URI=mongodb://127.0.0.1:27017/openhack

[Service]MONGO_DB=openhack

Type=simpleJWT_SECRET=supersecret

ExecStart={{.BinaryPath}} \PORT_RANGE_START=20000

    --deployment {{.Deployment}} \PORT_RANGE_END=29999

    --port {{.Port}} \BASE_PATH=/var/openhack

    --env-root {{.EnvRoot}} \```

    --app-version {{.Version}}

WorkingDirectory=/var/hypervisorRequired keys in backend per-deployment env files (from the confirmed decisions):

Restart=always

RestartSec=1s- `PREFORK`

StartLimitIntervalSec=30- `MONGO_URI`

StartLimitBurst=10- `JWT_SECRET`

StandardOutput=journal- `BADGE_PILES`

StandardError=journal

Environment="PATH=/usr/local/bin:/usr/bin:/bin"No other keys are required by default; deployment metadata (BIN/DEPLOYMENT/PORT/ENV_ROOT/APP_VERSION) is stored in the deployment MongoDB document and encoded in systemd ExecStart args.

Environment=GODEBUG=madvdontneed=1

---

[Install]

WantedBy=multi-user.target## 13. Status codes

````

| Code | Meaning |

**Rendering:**| ----------- | ------------------------------------------ |

- `{{.BinaryPath}}` → `/var/hypervisor/builds/v25.10.19.0`| 200/201/204 | success |

- `{{.Deployment}}` → `prod` (default)| 400 | invalid input |

- `{{.Port}}` → `8080` (default)| 401/403 | unauthorized |

- `{{.EnvRoot}}` → `/var/hypervisor/env`| 404 | not found |

- `{{.Version}}` → `v25.10.19.0`| 409 | conflict (duplicate stage, active deployment, etc.) |

| 412 | tests not passed but required |

**Dependencies:**| 422 | build/test/provision failed |

- Requires Redis service to be running (`After=network.target redis.service`, `Requires=redis.service`).| 500 | internal error |

- Restarts automatically with backoff (1s initial delay, up to 10 restarts in 30s interval).

---

---

## 14. Behavioral sequence

## 5. Build & Test Conventions

1. GitHub webhook hits `/hypervisor/hooks/github` (future feature).

### BUILD → Hypervisor clones repo, logs commit, builds if tag, and records `releases`. No automatic staging/promotion occurs.

2. Operator creates a stage via `POST /stages` supplying `{ releaseId, envTag }`.

```  → Hypervisor clones the release into`/var/openhack/repos/<stageId>`(one checkout per stage), copies`/var/openhack/env/template/.env`, persists a `stage`with status`pre`, and returns the template env text.

./BUILD [--output DIR]3. Operator edits the env locally, then pushes it with `PUT /stages/:stageId/env`.

# Example: → Hypervisor writes the `.env` file under `/var/openhack/env/<stageId>/.env`, transitions status to `active`, and clears any stale test markers.

./BUILD --output /var/hypervisor/builds4. Operator may start tests at any time with `POST /stages/:stageId/tests`.

```  → Hypervisor runs`./TEST`against the stage checkout, streams output over`/.ws/stages/:stageId/tests/:resultId` *(WebSocket emitting JSON payloads with `type` = `info`|`log`|`error`)*, and records a `stage_test_result` referencing the stage. Tests are manual; updating the env does **not** auto-run tests.

5. When ready, operator promotes the stage via `POST /deployments` (payload includes `stageId`).

- Reads local `VERSION` file. → Hypervisor ensures the stage is `active`, runs `./BUILD` if required, allocates a port, writes `/var/openhack/env/<stageId>/.env`, renders a systemd unit, starts the service, writes a `deployment` document pointing back to the stage, and marks the stage status `promoted`.

- Builds executable at `${OUTPUT}/${VERSION}` (e.g. `/var/hypervisor/builds/v25.10.19.0`).6. Promotion to “main” continues to use `POST /deployments/:deploymentId/promote`, updating the routing configuration so `/` maps to that deployment.

- Exit `0` on success.7. Further env tweaks repeat steps 3–4 on the same stage; after each successful test or review the operator may redeploy or create a new deployment from the updated stage.

8. Shutdown/Delete behavior matches the deployment lifecycle (stop unit, free port, remove files, mark deployment status).

### TEST9. Bulk sync (`POST /releases/sync`) reconciles commits/releases but leaves stages/deployments untouched.

10. Every action routes through `internal/events`, emitting structured entries into MongoDB `events` for auditing.

```

./TEST [--env-root PATH] [--app-version VERSION]---

# Example:

./TEST --env-root /var/openhack/env/v25.10.17.0-dev --app-version v25.10.17.0-dev## 15. Design principles

```

- One **binary per release**, multiple **stages** (and deployments) per version.

- Reads env from `${PATH}/.env`.- Parallel environments (dev/test/prod/judge) remain isolated.

- Prints logs to stdout/stderr.- Deterministic builds, env-first staging.

- Exit `0` on success.- Tests are operator-driven and reproducible; results are stored alongside the triggering stage.

- Tests run only when explicitly started through the stage testing API; creating or updating a stage does not auto-run tests.- Everything observable via event log.

- Self-contained lifecycle: release → stage → env update → (optional) test → deployment → promote → retire.

---- Full Swagger documentation and REST interface under `/hypervisor`.

- Designed for automated control from a Vercel dashboard and manual hyperuser intervention when needed.

## 6. Hypervisor Executable Runtime Flags

```
--deployment <envTag>
--port <port>
--env-root </var/hypervisor/env>
--app-version <vTAG>
```

---

## 7. MongoDB Collections

### releases

```json
{
  "id": "v25.10.17.0",
  "sha": "abc123",
  "createdAt": "2025-10-17T13:00:00Z"
}
```

### stages

```json
{
  "id": "v25.10.17.0-dev",
  "releaseId": "v25.10.17.0",
  "envTag": "dev",
  "status": "pre|active|promoted",
  "lastTestResultId": "tr_7yQ",
  "createdAt": "...",
  "updatedAt": "..."
}
```

### stage_test_results

```json
{
  "id": "tr_7yQ",
  "stageId": "v25.10.17.0-dev",
  "status": "running|passed|failed|canceled|error",
  "wsToken": "...",
  "logPath": "/var/openhack/runtime/logs/tr_7yQ.log",
  "startedAt": "...",
  "finishedAt": null
}
```

### deployments

```json
{
  "id": "v25.10.17.0-dev",
  "stageId": "v25.10.17.0-dev",
  "version": "v25.10.17.0",
  "envTag": "dev",
  "port": 20037,
  "status": "staged|ready|stopped|deleted",
  "createdAt": "...",
  "promotedAt": null
}
```

---

## 8. Event System

All operations emit structured events for auditing and replay. Events are persisted to the MongoDB `events` collection by the asynchronous emitter in `internal/events`.

### Event Schema

```go
type Event struct {
    TimeStamp  time.Time      `json:"timestamp" bson:"timestamp"`
    Action     string         `json:"action" bson:"action"`
    ActorID    string         `json:"actorID" bson:"actorID"`
    ActorRole  string         `json:"actorRole" bson:"actorRole"`   // "hyperuser", "system"
    TargetID   string         `json:"targetID" bson:"targetID"`
    TargetType string         `json:"targetType" bson:"targetType"` // "stage", "deployment"
    Props      map[string]any `json:"props" bson:"props"`
    Key        string         `json:"key,omitempty" bson:"key,omitempty"`
}
```

### Emitter Behaviour

- `events.Em` is initialized during application bootstrap once Mongo connectivity is available.
- Events are buffered through an internal channel (default capacity 1000) and flushed in batches of 50 via `InsertMany`.
- Timer ensures periodic flushes (2 s in normal deployments, 50 ms in `test` profile).
- If buffer is saturated, falls back to synchronous `InsertOne`.
- On shutdown `Emitter.Close()` drains the buffer to avoid data loss.
- Convenience wrappers (`internal/events/*.go`) set consistent actor/target constants.

### Example Event

```json
{
  "timestamp": "2025-10-19T12:34:56+02:00",
  "action": "stage.env_updated",
  "actorID": "user_123",
  "actorRole": "hyperuser",
  "targetID": "v25.10.17.0-dev",
  "targetType": "stage",
  "props": {}
}
```

---

## 9. Hypervisor REST API (all under `/hypervisor`)

| Endpoint                                              | Description                                            |
| ----------------------------------------------------- | ------------------------------------------------------ |
| `GET /hypervisor/meta/ping`                           | Hypervisor health probe (returns "PONG")               |
| `GET /hypervisor/meta/version`                        | Running hypervisor version string                      |
| `GET /hypervisor/hyperusers/whoami`                   | Authenticated hyperuser profile (JWT required)         |
| `GET /hypervisor/releases`                            | List all releases                                      |
| `POST /hypervisor/stages`                             | Create a pre-stage from a release and environment tag  |
| `GET /hypervisor/stages`                              | List all stages with status                            |
| `GET /hypervisor/stages/:stageId`                     | Inspect a single stage                                 |
| `GET /hypervisor/stages/:stageId/env`                 | Read the stage `.env` contents from disk               |
| `PUT /hypervisor/stages/:stageId/env`                 | Update the stage `.env` file (pre → active transition) |
| `POST /hypervisor/stages/:stageId/tests`              | Start a manual test run for the stage                  |
| `GET /.ws/stages/:stageId/tests/:resultId`            | WebSocket stream for stage test logs (JSON messages)   |
| `POST /hypervisor/deployments`                        | Promote an active stage to a deployment                |
| `GET /hypervisor/deployments`                         | List active deployments                                |
| `POST /hypervisor/deployments/:deploymentId/promote`  | Promote a deployment to main route                     |
| `POST /hypervisor/deployments/:deploymentId/shutdown` | Stop deployment gracefully                             |
| `DELETE /hypervisor/deployments/:deploymentId`        | Delete deployment (`?force=true` stops first)          |
| `GET /hypervisor/env/template`                        | Read the canonical environment template                |
| `PUT /hypervisor/env/template`                        | Update the canonical environment template              |
| `POST /hypervisor/releases/sync`                      | Bulk sync commits and releases from GitHub             |
| `GET /hypervisor/docs`                                | Swagger UI                                             |
| `GET /hypervisor/docs/doc.json`                       | Swagger spec (JSON)                                    |

**Authentication:**

- All routes (including WebSocket upgrades) require the Hyperuser JWT in the `Authorization: Bearer <token>` header.

---

## 10. Project Structure

```
openhack-hypervisor/
├─ cmd/
│  ├─ server/
│  │  └─ main.go                 # Fiber app bootstrap + /hypervisor routing
│  └─ hyperctl/
│     └─ main.go                 # CLI entry point
├─ internal/
│  ├─ api/                       # HTTP handlers + swagger annotations
│  │  ├─ staging.go
│  │  ├─ releases.go
│  │  └─ env.go
│  ├─ core/                      # Business logic
│  │  ├─ stage.go
│  │  ├─ sync.go
│  │  └─ env_template.go
│  ├─ db/                        # Mongo + Redis initialization
│  │  └─ db.go
│  ├─ env/                       # Runtime flags + env loading
│  │  └─ env.go
│  ├─ events/                    # Event emitter + wrappers
│  │  ├─ emitter.go
│  │  ├─ emit.go
│  │  └─ *_wrappers.go
│  ├─ hyperusers/                # Authentication
│  │  ├─ login_handlers.go
│  │  └─ routes.go
│  ├─ hyperctl/                  # CLI tooling
│  │  ├─ commands/
│  │  ├─ build/
│  │  ├─ fs/
│  │  ├─ git/
│  │  ├─ health/
│  │  ├─ state/
│  │  ├─ system/
│  │  ├─ systemd/
│  │  └─ testing/
│  ├─ models/                    # Mongo persistence models
│  │  ├─ stage.go
│  │  ├─ deployment.go
│  │  ├─ release.go
│  │  ├─ stage_test_result.go
│  │  ├─ event.go
│  │  └─ hyperuser.go
│  ├─ paths/                     # Filesystem path constants
│  │  └─ paths.go
│  ├─ swagger/                   # Embedded Swagger assets
│  │  ├─ routes.go
│  │  ├─ tags.go
│  │  ├─ docs_embed.go
│  │  └─ docs/
│  └─ utils/                     # Helpers
│     ├─ errors.go
│     ├─ locals.go
│     └─ ids.go
├─ test/
│  ├─ hyperusers/
│  │  ├─ login_test.go
│  │  └─ main_test.go
│  └─ sync/
│     ├─ main_test.go
│     └─ sync_test.go
├─ API_SPEC                      # Generate swagger docs
├─ BUILD                         # Build script
├─ TEST                          # Test script
├─ RUNDEV                        # Development bootstrap
├─ RELEASE                       # Release process
├─ VERSION                       # Version file
├─ go.mod / go.sum
└─ HYPERVISOR_DESIGN.md          # This document
```

---

## 11. Filesystem Paths (via `internal/paths/paths.go`)

```go
const (
    HypervisorBaseDir   = "/var/hypervisor"
    HypervisorReposDir  = "/var/hypervisor/repos"
    HypervisorBuildsDir = "/var/hypervisor/builds"
    HypervisorEnvDir    = "/var/hypervisor/env"
    HypervisorLogsDir   = "/var/hypervisor/logs"

    OpenHackBaseDir  = "/var/openhack"
    OpenHackReposDir = "/var/openhack/repos"
    OpenHackBuildsDir= "/var/openhack/builds"
    OpenHackEnvDir   = "/var/openhack/env"

    SystemdUnitDir   = "/lib/systemd/system"
)
```

---

## 12. Environment Variables

### Hypervisor Runtime

```
HTTP_ADDR=:8080                         # Hypervisor listen address
MONGO_URI=mongodb://127.0.0.1:27017     # MongoDB connection
MONGO_DB=openhack                       # Database name
JWT_SECRET=<secret>                     # Authentication secret
REDIS_URI=redis://127.0.0.1:6379        # Redis connection
REPO_URL=https://github.com/OpenLabsRo/openhack-backend  # Backend repo
```

### Per-Deployment Backend Env (from `/var/openhack/env/<stageId>/.env`)

Required keys:

- `PREFORK` — Enable Go fiber prefork mode
- `MONGO_URI` — Backend MongoDB connection
- `JWT_SECRET` — Backend authentication secret
- `BADGE_PILES` — Game configuration

Additional keys may be provided as needed by the backend application.

---

## 13. HTTP Status Codes

| Code        | Meaning                              |
| ----------- | ------------------------------------ |
| 200/201/204 | Success                              |
| 400         | Invalid input                        |
| 401/403     | Unauthorized                         |
| 404         | Not found                            |
| 409         | Conflict (duplicate stage, etc.)     |
| 412         | Precondition failed (tests required) |
| 422         | Build/test/provision failed          |
| 500         | Internal error                       |

---

## 14. Behavioral Workflow

1. **Initial Setup (Operator runs `hyperctl setup`)**

   - Prerequisites verified, directories created.
   - Hypervisor repository cloned and built.
   - Systemd unit installed and service started.
   - Health check confirms readiness.

2. **Create Stage (POST /stages)**

   - Operator submits `{releaseId, envTag}`.
   - Hypervisor clones backend release, seeds template env on disk.
   - Stage created with status `pre`.
   - Returns template environment text.

3. **Update Stage Environment (PUT /stages/:stageId/env)**

   - Operator edits and submits updated `.env` file.
   - File written to `/var/openhack/env/<stageId>/.env`.
   - Stage transitions from `pre` → `active`.

4. **Optional: Test Stage (POST /stages/:stageId/tests)**

   - Operator starts manual test run.
   - Backend test script (`./TEST`) executed against stage checkout.
   - Output streamed over WebSocket.
   - Result recorded as `stage_test_result`.

5. **Deploy Stage (POST /deployments)**

   - Operator submits active `stageId`.
   - Hypervisor builds backend (if needed), allocates port.
   - Systemd unit for backend created and started.
   - Deployment record created with status `ready`.
   - Stage marked `promoted`.

6. **Promote to Main (POST /deployments/:deploymentId/promote)**

   - Operator designates deployment as main.
   - Routing updated so `/` proxies to that deployment.

7. **Iterate or Retire**

   - Further env edits repeat steps 3–4.
   - Deployments can be stopped or deleted.
   - All actions emit events for audit log.

8. **Shutdown (Operator runs `hyperctl nagasaki`)**

   - Hypervisor service stopped gracefully.
   - Backend deployments continue running.

9. **Complete Cleanup (Operator runs `hyperctl hiroshima`)**
   - All systemd units disabled and removed.
   - All hypervisor and openhack directories deleted.
   - System returned to pristine state.

---

## 15. Design Principles

- **Dual Layer:** Hypervisor daemon (manages backend deployments) + CLI tooling (manages hypervisor itself).
- **One binary per release:** Build artifact is immutable and reused across stages and deployments.
- **Multiple parallel environments:** dev/test/prod/judge remain isolated with separate stages, deployments, and ports.
- **Deterministic builds:** Env-first approach ensures reproducibility.
- **Manual operator control:** Tests and deployments are explicitly triggered, never automatic.
- **Full observability:** All operations emit structured events for auditing.
- **Self-contained lifecycle:** release → stage → env → test → deployment → promote → retire.
- **REST API + CLI:** Web dashboard integration + local command-line automation.
- **Systemd-native:** Deployments managed via systemd units for consistency with Linux ecosystem.

---

## 16. Security Considerations

- Keep secrets (JWT_SECRET, MONGO_URI, REDIS credentials) in `.env` files; never commit to repository.
- Hyperctl operations (especially `hiroshima`) require `sudo` for systemd access; restrict to trusted operators.
- All Hypervisor API routes requiring state changes enforce JWT Bearer authentication.
- WebSocket connections for test streams authenticate via token query parameter.
- Sensitive logs are persisted to `/var/hypervisor/logs/` with appropriate file permissions.
