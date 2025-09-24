package events

import "hypervisor/internal/models"

func (e *Emitter) HyperUserLogin(hyperuserID string) {
	evt := models.Event{
		Action: "hyperuser.login",

		ActorRole: ActorHyperUser,
		ActorID:   hyperuserID,

		TargetType: TargetHyperUser,
		TargetID:   hyperuserID,

		Props: nil,
	}

	e.Emit(evt)
}
