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
