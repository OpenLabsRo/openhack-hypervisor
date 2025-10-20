package helpers

import (
	"encoding/json"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func API_CreateStage(
	t *testing.T,
	app *fiber.App,
	token string,
	releaseID string,
	envTag string,
) (bodyBytes []byte, statusCode int) {
	// payload for create stage request
	payload := struct {
		ReleaseID string `json:"releaseId"`
		EnvTag    string `json:"envTag"`
	}{
		ReleaseID: releaseID,
		EnvTag:    envTag,
	}

	// marshalling the payload into JSON
	sendBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	return RequestRunner(t, app,
		"POST",
		"/hypervisor/stages",
		sendBytes,
		&token,
	)
}

func API_ListStages(
	t *testing.T,
	app *fiber.App,
	token string,
) (bodyBytes []byte, statusCode int) {
	return RequestRunner(t, app,
		"GET",
		"/hypervisor/stages",
		nil,
		&token,
	)
}

func API_GetStage(
	t *testing.T,
	app *fiber.App,
	token string,
	stageID string,
) (bodyBytes []byte, statusCode int) {
	return RequestRunner(t, app,
		"GET",
		"/hypervisor/stages/"+stageID,
		nil,
		&token,
	)
}

func API_StartTestRun(
	t *testing.T,
	app *fiber.App,
	token string,
	stageID string,
) (bodyBytes []byte, statusCode int) {
	return RequestRunner(t, app,
		"POST",
		"/hypervisor/stages/"+stageID+"/tests",
		nil,
		&token,
	)
}

func API_DeleteStage(
	t *testing.T,
	app *fiber.App,
	token string,
	stageID string,
) (bodyBytes []byte, statusCode int) {
	return RequestRunner(t, app,
		"DELETE",
		"/hypervisor/stages/"+stageID,
		nil,
		&token,
	)
}
