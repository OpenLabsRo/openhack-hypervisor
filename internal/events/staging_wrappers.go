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

// StageEnvUpdated records that the stage environment has been updated on disk.
func (e *Emitter) StageEnvUpdated(stage models.Stage) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.env_updated",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   stage.ID,
		TargetType: "stage",
	}

	e.Emit(evt)
}

// StageDeleted records the deletion of a stage and its resources.
func (e *Emitter) StageDeleted(stageID string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "stage.deleted",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   stageID,
		TargetType: "stage",
	}

	e.Emit(evt)
}

// TestStarted records the beginning of a manual test run.
func (e *Emitter) TestStarted(stage models.Stage, test models.Test) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "test.started",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   test.ID,
		TargetType: "test",
		Props: map[string]any{
			"stageId": stage.ID,
		},
	}

	e.Emit(evt)
}

// TestPassed records a successful test completion.
func (e *Emitter) TestPassed(stageID, testID string, duration time.Duration) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "test.passed",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   testID,
		TargetType: "test",
		Props: map[string]any{
			"stageId":  stageID,
			"duration": duration.Seconds(),
		},
	}

	e.Emit(evt)
}

// TestFailed records a failing test completion.
func (e *Emitter) TestFailed(stageID, testID string, errMsg string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "test.failed",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   testID,
		TargetType: "test",
		Props: map[string]any{
			"stageId": stageID,
			"error":   errMsg,
		},
	}

	e.Emit(evt)
}

// TestCanceled records a canceled test.
func (e *Emitter) TestCanceled(stageID, testID string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "test.canceled",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   testID,
		TargetType: "test",
		Props: map[string]any{
			"stageId": stageID,
		},
	}

	e.Emit(evt)
}
