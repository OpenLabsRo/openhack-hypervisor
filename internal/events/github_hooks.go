package events

import "hypervisor/internal/models"

// targetGitCommit marks events emitted from GitHub webhook ingestion.
const targetGitCommit = "git_commit"

// GitHubCommitReceived records an event for each persisted webhook commit.
func (e *Emitter) GitHubCommitReceived(deliveryID, sha, ref, message string) {
	if e == nil {
		return
	}

	evt := models.Event{
		Action: "github.commit.received",

		ActorRole: ActorSystem,
		ActorID:   deliveryID,

		TargetType: targetGitCommit,
		TargetID:   sha,

		Props: map[string]any{
			"ref":     ref,
			"sha":     sha,
			"message": message,
		},
	}

	e.Emit(evt)
}
