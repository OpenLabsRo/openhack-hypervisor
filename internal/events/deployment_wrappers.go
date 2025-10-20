package events

import "hypervisor/internal/models"

// DeploymentCreated records a successful staged deployment record creation.
func (e *Emitter) DeploymentCreated(dep models.Deployment) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "deployment.created",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   dep.ID,
		TargetType: "deployment",
		Props: map[string]any{
			"status":  dep.Status,
			"stageId": dep.StageID,
		},
	}

	e.Emit(evt)
}

// DeploymentCreateFailed records a failure while creating a deployment record.
func (e *Emitter) DeploymentCreateFailed(deploymentID string, err error) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "deployment.create_failed",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   deploymentID,
		TargetType: "deployment",
		Props: map[string]any{
			"error": err.Error(),
		},
	}

	e.Emit(evt)
}

// DeploymentPromoted records a deployment being promoted to main.
func (e *Emitter) DeploymentPromoted(dep models.Deployment) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "deployment.promoted",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   dep.ID,
		TargetType: "deployment",
		Props: map[string]any{
			"stageId": dep.StageID,
		},
	}

	e.Emit(evt)
}

// DeploymentStopped records a deployment being stopped.
func (e *Emitter) DeploymentStopped(dep models.Deployment) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "deployment.stopped",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   dep.ID,
		TargetType: "deployment",
		Props: map[string]any{
			"stageId": dep.StageID,
		},
	}

	e.Emit(evt)
}

// DeploymentDeleted records a deployment being deleted.
func (e *Emitter) DeploymentDeleted(dep models.Deployment) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action:     "deployment.deleted",
		ActorID:    ActorSystem,
		ActorRole:  ActorSystem,
		TargetID:   dep.ID,
		TargetType: "deployment",
		Props: map[string]any{
			"stageId": dep.StageID,
		},
	}

	e.Emit(evt)
}
