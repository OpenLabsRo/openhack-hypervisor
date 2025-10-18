package events

import (
	"fmt"
	"time"

	"hypervisor/internal/models"
)

// StagePrepared records the successful creation of a stage.
func (e *Emitter) StagePrepared(stage models.Stage) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.prepared",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   stage.ID,
		TargetType: "stage",
		Props: map[string]any{
			"releaseId": stage.ReleaseID,
			"envTag":    stage.EnvTag,
		},
	}

	e.Emit(evt)
}

// StageFailed records a stage bootstrap failure.
func (e *Emitter) StageFailed(releaseID, envTag string, err error) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.failed",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   releaseID,
		TargetType: "release",
		Props: map[string]any{
			"envTag": envTag,
			"error":  fmt.Sprint(err),
		},
	}

	e.Emit(evt)
}

// StageSessionCreated records submission of a new stage session.
func (e *Emitter) StageSessionCreated(stage models.Stage, session models.StageSession) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.session_created",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   session.ID,
		TargetType: "stage_session",
		Props: map[string]any{
			"stageId":   stage.ID,
			"releaseId": stage.ReleaseID,
			"envTag":    stage.EnvTag,
		},
	}

	e.Emit(evt)
}

// StageEnvUpdated records that the stage environment has been updated on disk.
func (e *Emitter) StageEnvUpdated(stage models.Stage, session models.StageSession) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.env_updated",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   stage.ID,
		TargetType: "stage",
		Props: map[string]any{
			"sessionId": session.ID,
		},
	}

	e.Emit(evt)
}

// StageTestStarted records the beginning of a manual stage test run.
func (e *Emitter) StageTestStarted(stage models.Stage, session models.StageSession, result models.StageTestResult) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.test_started",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   result.ID,
		TargetType: "stage_test_result",
		Props: map[string]any{
			"stageId":   stage.ID,
			"sessionId": session.ID,
		},
	}

	e.Emit(evt)
}

// StageTestPassed records a successful stage test completion.
func (e *Emitter) StageTestPassed(stageID, sessionID, resultID string, duration time.Duration) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.test_passed",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   resultID,
		TargetType: "stage_test_result",
		Props: map[string]any{
			"stageId":   stageID,
			"sessionId": sessionID,
			"duration":  duration.Seconds(),
		},
	}

	e.Emit(evt)
}

// StageTestFailed records a failing stage test completion.
func (e *Emitter) StageTestFailed(stageID, sessionID, resultID string, errMsg string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.test_failed",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   resultID,
		TargetType: "stage_test_result",
		Props: map[string]any{
			"stageId":   stageID,
			"sessionId": sessionID,
			"error":     errMsg,
		},
	}

	e.Emit(evt)
}

// StageTestCanceled records a canceled stage test.
func (e *Emitter) StageTestCanceled(stageID, sessionID, resultID string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.test_canceled",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   resultID,
		TargetType: "stage_test_result",
		Props: map[string]any{
			"stageId":   stageID,
			"sessionId": sessionID,
		},
	}

	e.Emit(evt)
}
