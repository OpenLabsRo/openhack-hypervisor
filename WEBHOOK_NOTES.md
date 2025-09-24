# GitHub Webhook Implementation Notes

- Endpoint: `POST /hypervisor/github/commits`
- Validate `X-Hub-Signature-256` using `GITHUB_WEBHOOK_SECRET`.
- Accept only `push` events; return 204 for others.
- Require `X-GitHub-Delivery`; use `(delivery_id, commit_sha)` for idempotency.
- Persist each commit in `hypervisor.git_commits` with fields: `delivery_id`, `ref`, `sha`, `message`, `author{name,email}`, `timestamp`.
- Emit `events.Em.Emit` with action `github.commit.received` after storing.
- No full payload storage; repository name implicit.
- Unit-test handler with mocked collection: signature success/failure, non-push short-circuit, DB write.
- Update env loader for `GITHUB_WEBHOOK_SECRET` and DB init to expose `GitCommits` collection.
- Register routes in `internal/app.go` and reuse existing emitter config (`deployment` toggles flush speed).
