# ⚙️ OpenHack Hypervisor — Final Contract Specification

**Version:** 2025-10-17  
**Purpose:** automate build, test, deploy, routing, and lifecycle management of OpenHack backend services with full control from a web interface.  
**Tech stack:** Go (Fiber + MongoDB) + systemd.  
**Public base URL:** `https://DOMAIN/hypervisor/*`

---

## 1. Core concepts

- **Release** — immutable build artifact of a tagged commit.
- **Stage** — configuration workspace (`version + envTag`) that owns the environment file, repo clone, test history, and lifecycle status (`pre`, `active`, `promoted`).
- **Stage session** — immutable snapshot of an env submission to a stage; sessions accumulate history and reference the stage they belong to.
- **Stage test result** — outcome of a manual test run (started via API, streamed over WebSocket) linked to both the stage and the session that triggered it.
- **Deployment** — running instance of a stage, uniquely identified by `vVERSION-ENVTAG` (e.g. `v25.10.17.0-prod`); deployments are created only from `active` stages.
- **Event** — structured log entry emitted on every operation (for auditing and correlation).
- **Routing** — URL prefixes mapped to deployments; `/` points to the promoted “main” one.
- **Hypervisor API** — all control routes served under `/hypervisor`.

---

## 2. Filesystem layout

```
/var/openhack/
  repos/
    v25.10.17.0/               # cloned at tag
  builds/
    v25.10.17.0                # compiled binary
  env/
    template/.env              # canonical staging template
    v25.10.17.0-dev/.env
    v25.10.17.0-prod/.env
  runtime/
    logs/
    tmp/

/etc/openhack/versions/
  v25.10.17.0-dev/env          # BIN/DEPLOYMENT/PORT/ENV_ROOT/APP_VERSION + user env
```

Per-deployment environment files live under `/var/openhack/env`. There is a single canonical template at `/var/openhack/env/template/.env`. Each deployment uses exactly one environment file derived from the template; the file is located under `/var/openhack/env/<deployment-id>/.env`.

---

## 3. Build & test conventions

### BUILD

```
./BUILD [--output DIR]
# Example:
./BUILD --output /var/openhack/builds
```

- Reads local `VERSION` file.
- Builds executable at `${OUTPUT}/${VERSION}` (e.g. `/var/openhack/builds/v25.10.17.0`).
- Exit `0` on success.

### TEST

```
./TEST [--env-root PATH] [--app-version VERSION]
# Example:
./TEST --env-root /var/openhack/env/v25.10.17.0-dev --app-version v25.10.17.0-dev
```

- Reads env from `${PATH}/.env`.
- Prints logs to stdout/stderr.
- Exit `0` on success.
- Tests run only when explicitly started through the stage testing API; creating or updating a stage does not auto-run tests.

---

## 4. Executable runtime flags

```
--deployment <envTag>
--port <port>
--env-root </var/openhack/env/vTAG-ENV>
--app-version <vTAG-ENV>
```

---

## 5. systemd integration

Unit: `openhack-backend@.service` (installed into the systemd unit directory defined in the codebase via `paths.SystemdUnitDir`)

```
[Unit]
Description=OpenHack backend %i
After=network-online.target
Wants=network-online.target

[Service]
Type=exec

# The environment file for an individual deployment lives at /var/openhack/env/<deployment-id>/.env
EnvironmentFile=/var/openhack/env/%i/.env
ExecStart=${BIN} \
 --deployment ${DEPLOYMENT} \
 --port ${PORT} \
 --env-root ${ENV_ROOT} \
 --app-version ${APP_VERSION}
Restart=always
RestartSec=1
DynamicUser=yes
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=read-only
PrivateTmp=yes
StateDirectory=openhack-%i
RuntimeDirectory=openhack-%i

[Install]
WantedBy=multi-user.target
```

Environment example:

```
BIN=/var/openhack/builds/v25.10.17.0
DEPLOYMENT=dev
PORT=20037
ENV_ROOT=/var/openhack/env/v25.10.17.0-dev
APP_VERSION=v25.10.17.0-dev
# --- begin user env ---
PREFORK=false
MONGO_URI=mongodb+srv://...
JWT_SECRET=abcd
BADGE_PILES=6
# --- end user env ---
```

---

## 6. MongoDB collections

### releases

```
{
  "id": "v25.10.17.0",
  "sha": "abc123",
  "artifact": { "binPath": "/var/openhack/builds/v25.10.17.0" },
  "createdAt": "2025-10-17T13:00:00Z"
}
```

### stages

```
{
  "id": "v25.10.17.0-dev",
  "releaseId": "v25.10.17.0",
  "envTag": "dev",
  "status": "pre|active|promoted",
  "envText": "PREFORK=false\n...",
  "repoPath": "/var/openhack/repos/v25.10.17.0",
  "latestSessionId": "ssn_Qa7",
  "createdAt": "...",
  "updatedAt": "...",
  "lastTestResultId": "tr_7yQ"    # nullable; most recent manual test run
}
```

### stage_sessions

```
{
  "id": "ssn_Qa7",
  "stageId": "v25.10.17.0-dev",
  "envText": "PREFORK=false\n...",
  "author": "user_123",
  "notes": "Adjusted Redis host",
  "source": "template|manual|import",
  "createdAt": "...",
  "testResultId": "tr_7yQ"       # nullable reference to the test run started from this session
}
```

### stage_test_results

```
{
  "id": "tr_7yQ",
  "stageId": "v25.10.17.0-dev",
  "sessionId": "ssn_Qa7",
  "status": "running|passed|failed|canceled|error",
  "wsToken": "...",
  "logPath": "/var/openhack/runtime/logs/tr_7yQ.log",
  "startedAt": "...",
  "finishedAt": null
}
```

### deployments

```
{
  "id": "v25.10.17.0-dev",
  "stageId": "v25.10.17.0-dev",
  "version": "v25.10.17.0",
  "envTag": "dev",
  "port": 20037,                  # nullable; staged deployments have no port yet
  "status": "staged|ready|stopped|deleted",
  "createdAt": "...",
  "promotedAt": null
}
```

---

## 7. Event system

All operations emit structured events for auditing and replay. Events are persisted to the MongoDB `events` collection by the asynchronous emitter in `internal/events`.

### Event schema

The canonical shape is `internal/models.Event`:

```
type Event struct {
    TimeStamp  time.Time         `json:"timestamp" bson:"timestamp"`
    Action     string            `json:"action" bson:"action"`
    ActorID    string            `json:"actorID" bson:"actorID"`
    ActorRole  string            `json:"actorRole" bson:"actorRole"`   // e.g. "hyperuser", "system"
    TargetID   string            `json:"targetID" bson:"targetID"`
    TargetType string            `json:"targetType" bson:"targetType"` // e.g. "stage", "deployment"
    Props      map[string]any    `json:"props" bson:"props"`
    Key        string            `json:"key,omitempty" bson:"key,omitempty"`
}
```

Before persistence the emitter stamps every event with the `Europe/Bucharest` timezone so audit timelines align with operations.

### Emitter behaviour

- `events.Em` is initialised during application bootstrap once Mongo connectivity is available.
- Events are buffered through an internal channel (default capacity 1000) and flushed in batches of 50 via `InsertMany`. A timer (2 s in normal deployments, 50 ms in the `test` profile) ensures periodic flushes even when the batch is not full.
- If the buffer is saturated the emitter falls back to a synchronous `InsertOne`.
- On shutdown `Emitter.Close()` drains the buffer to avoid data loss.
- Convenience wrappers (`internal/events/*.go`) set consistent actor/target constants such as `ActorHyperUser`, `ActorSystem`, ensuring consumers see predictable `actorRole`/`targetType` values.

Example event:

```
{
  "timestamp": "2025-10-17T12:34:56+02:00",
  "action": "stage.session_created",
  "actorID": "user_123",
  "actorRole": "hyperuser",
  "targetID": "v25.10.17.0-dev",
  "targetType": "stage",
  "props": { "sessionId": "ssn_Qa7" }
}
```

Typical events include:

- `stage.prepared`, `stage.session_created`, `stage.env_updated`, `stage.failed`
- `stage.test_started`, `stage.test_passed`, `stage.test_failed`, `stage.test_canceled`
- `deployment.created`, `deployment.create_failed`
- `deploy.started|ready|health_failed|stopped|deleted`
- `promote.changed`
- `sync.started|release_created|failed|finished`

All code paths must emit via `events.Em.Emit(...)` (or the provided wrappers) to benefit from batching and schema enforcement.

---

## 8. API (all under `/hypervisor`)

| Endpoint                                       | Description                                                                                                                                         |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET /hypervisor/meta/ping`                 | Hypervisor health probe (PONG)                                                                                                                      |
| `GET /hypervisor/meta/version`              | Running hypervisor version string                                                                                                                   |
| `GET /hypervisor/hyperusers/whoami`          | Return the authenticated hyperuser profile (JWT required)                                                                                            |
| `GET /releases`                              | List all releases                                                                                                                                    |
| `POST /stages`                                 | Create a **pre-stage**: clones the release repo, seeds the template env, returns the stage (status `pre`) plus the template contents                 |
| `GET /stages`                                  | List stages with status, latest session info, and last test summary                                                                                 |
| `GET /stages/:stageId`                         | Inspect a single stage (current env, status timeline, last test results)                                                                            |
| `POST /stages/:stageId/sessions`               | Submit an env snapshot; first submission moves the stage from `pre` → `active`, later submissions append history                                    |
| `GET /stages/:stageId/sessions`                | List all sessions for the stage (immutable env history, newest first)                                                                               |
| `POST /stages/:stageId/tests`                  | Start a manual test run for the current env; response returns `{ testResultId, wsToken }` for streaming                                             |
| `POST /stages/:stageId/tests/:resultId/cancel` | Cancel an in-flight stage test (optional, when supported)                                                                                           |
| `GET /.ws/stages/:stageId/tests/:resultId`     | WebSocket for live `./TEST` output associated with the stage test                                                                                   |
| `POST /deployments`                            | Promote an `active` stage to a deployment (writes build artifacts + systemd unit, records deployment referencing `stageId`)                         |
| `GET /deployments`                             | List active deployments                                                                                                                             |
| `GET /deployments/main`                        | Get main deployment                                                                                                                                 |
| `POST /deployments/:deploymentId/promote`      | Promote an existing deployment to become the main route                                                                                             |
| `POST /deployments/:deploymentId/shutdown`     | Stop process, keep record                                                                                                                           |
| `DELETE /deployments/:deploymentId`            | Delete deployment (`?force=true` stops first)                                                                                                       |
| `GET /routes/main`                             | Get main route info                                                                                                                                 |
| `PUT /routes/main`                             | Set main deployment                                                                                                                                 |
| `POST /sync`                                   | Bulk sync commits and releases                                                                                                                      |
| `GET /hypervisor/docs` / `GET /hypervisor/docs/doc.json` | Swagger UI + generated spec                                                                                                         |

- All write routes use JWT Bearer auth.
- WebSocket auth via `?token=` query param.

---

## 9. Routing behavior

| URL prefix            | Target                         | Purpose         |
| --------------------- | ------------------------------ | --------------- |
| `/v25.10.17.0-dev/*`  | localhost:20037                | dev deployment  |
| `/v25.10.17.0-prod/*` | localhost:20055                | prod deployment |
| `/`                   | whichever deployment is “main” | public route    |

Multiple deployments of the same version coexist.

---

## 10. Swagger documentation

- Uses swaggo tooling; generated via `./API_SPEC` which wraps `swag init`.
- Output lives in `internal/swagger/docs/`.
- Served at `/hypervisor/docs` (HTML UI) and `/hypervisor/docs/doc.json` (raw spec).
- `BasePath: /hypervisor`
- Security: `BearerAuth` (Authorization header)

---

## 11. Project structure

```
openhack-hypervisor/
├─ cmd/server/
│  └─ main.go                 # fiber app, mount /hypervisor, swagger, routes
├─ internal/
│  ├─ api/                    # HTTP layer + swagger annotations (stages, deployments, tests, sync, hyperusers)
│  ├─ core/                   # business logic (stage lifecycle, deployments, tests, routing, sync)
│  ├─ db/                     # Mongo + Redis initialization
│  ├─ env/                    # runtime flags/env loading
│  ├─ events/                 # reusable event emitter
│  ├─ hyperusers/             # authentication handlers
│  ├─ models/                 # Mongo persistence models (stage, session, test result, deployment, release, etc.)
│  ├─ swagger/                # embedded swagger assets + UI routes
│  └─ utils/                  # helpers (errors, locals, ids)
├─ API_SPEC                   # helper script to regenerate swagger docs
├─ HYPERVISOR_DESIGN.md       # this contract
├─ go.mod / go.sum
└─ VERSION
```

---

## 12. Environment variables

These are the minimal environment variables consumed by the hypervisor and the build/test/deploy tooling. Backend deployments will additionally read their per-deployment `.env` from `/var/openhack/env/<deployment-id>/.env`.

```
HTTP_ADDR=:9090
MONGO_URI=mongodb://127.0.0.1:27017/openhack
MONGO_DB=openhack
JWT_SECRET=supersecret
PORT_RANGE_START=20000
PORT_RANGE_END=29999
BASE_PATH=/var/openhack
```

Required keys in backend per-deployment env files (from the confirmed decisions):

- `PREFORK`
- `MONGO_URI`
- `JWT_SECRET`
- `BADGE_PILES`

No other keys are required by default; deployment metadata (BIN/DEPLOYMENT/PORT/ENV_ROOT/APP_VERSION) is stored in the deployment MongoDB document and encoded in systemd ExecStart args.

---

## 13. Status codes

| Code        | Meaning                                    |
| ----------- | ------------------------------------------ |
| 200/201/204 | success                                    |
| 400         | invalid input                              |
| 401/403     | unauthorized                               |
| 404         | not found                                  |
| 409         | conflict (duplicate stage, active deployment, etc.) |
| 412         | tests not passed but required              |
| 422         | build/test/provision failed                |
| 500         | internal error                             |

---

## 14. Behavioral sequence

1. GitHub webhook hits `/hypervisor/hooks/github` (future feature).  
   → Hypervisor clones repo, logs commit, builds if tag, and records `releases`. No automatic staging/promotion occurs.
2. Operator creates a stage via `POST /stages` supplying `{ releaseId, envTag }`.  
   → Hypervisor clones the release into the stage workspace, copies `/var/openhack/env/template/.env`, persists a `stage` with status `pre`, and returns the template env text.
3. Operator edits the env locally, then submits it with `POST /stages/:stageId/sessions`.  
   → Hypervisor stores a `stage_session`, updates the stage’s current env, and transitions status to `active`. Subsequent submissions append more sessions without changing the stage id.
4. Operator may start tests at any time with `POST /stages/:stageId/tests`.  
   → Hypervisor runs `./TEST` against the stage checkout, streams output over `/.ws/stages/:stageId/tests/:resultId`, and records a `stage_test_result` referencing both the stage and the triggering session. Tests are manual; creating a session does **not** auto-run tests.
5. When ready, operator promotes the stage via `POST /deployments` (payload includes `stageId`).  
   → Hypervisor ensures the stage is `active`, runs `./BUILD` if required, allocates a port, writes `/var/openhack/env/<stageId>/.env`, renders a systemd unit, starts the service, writes a `deployment` document pointing back to the stage, and marks the stage status `promoted`.
6. Promotion to “main” continues to use `POST /deployments/:deploymentId/promote`, updating the routing configuration so `/` maps to that deployment.
7. Further env tweaks repeat steps 3–4 on the same stage; after each successful test or review the operator may redeploy or create a new deployment from the updated stage history.
8. Shutdown/Delete behavior matches the deployment lifecycle (stop unit, free port, remove files, mark deployment status).
9. Bulk sync (`POST /sync`) reconciles commits/releases but leaves stages/deployments untouched.
10. Every action routes through `internal/events`, emitting structured entries into MongoDB `events` for auditing.

---

## 15. Design principles

- One **binary per release**, multiple **stages** (and deployments) per version.
- Parallel environments (dev/test/prod/judge) remain isolated.
- Deterministic builds, env-first staging.
- Tests are operator-driven and reproducible; results are stored alongside the triggering stage session.
- Everything observable via event log.
- Self-contained lifecycle: release → stage → session → (optional) test → deployment → promote → retire.
- Full Swagger documentation and REST interface under `/hypervisor`.
- Designed for automated control from a Vercel dashboard and manual hyperuser intervention when needed.

