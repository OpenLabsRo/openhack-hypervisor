package events

import "hypervisor/internal/models"

// SyncStarted records the beginning of a repository sync operation.
func (e *Emitter) SyncStarted(repoURL string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "sync.started",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   repoURL,
		TargetType: "repository",
		Props:      map[string]any{},
	}

	e.Emit(evt)
}

// SyncFailed records a failure during repository sync.
func (e *Emitter) SyncFailed(repoURL string, err error) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "sync.failed",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   repoURL,
		TargetType: "repository",
		Props: map[string]any{
			"error": err.Error(),
		},
	}

	e.Emit(evt)
}

// SyncReleaseCreated records that a release has been created as part of a sync run.
func (e *Emitter) SyncReleaseCreated(tag, sha string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "release_created",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   tag,
		TargetType: "release",
		Props: map[string]any{
			"sha": sha,
		},
	}

	e.Emit(evt)
}

// SyncFinished records completion of a repository sync.
func (e *Emitter) SyncFinished(repoURL string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "sync.finished",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   repoURL,
		TargetType: "repository",
		Props:      map[string]any{},
	}

	e.Emit(evt)
}
