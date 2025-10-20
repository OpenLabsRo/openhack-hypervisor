package staging

import (
	"encoding/json"
	"net/http"
	"testing"

	"hypervisor/internal/models"
	"hypervisor/test/helpers"

	"github.com/stretchr/testify/require"
)

func TestCreateAndListStage(t *testing.T) {
	// Login to get a token
	body, statusCode := helpers.API_HyperUsersLogin(t, app, "testhyperuser", "testhyperuser")
	require.Equal(t, http.StatusOK, statusCode)

	var payload struct {
		Token string `json:"token"`
	}
	err := json.Unmarshal(body, &payload)
	require.NoError(t, err)

	// Sync releases
	_, statusCode = helpers.API_SyncReleases(t, app, payload.Token)
	require.Equal(t, http.StatusOK, statusCode)

	// Create a release to associate with the stage
	releaseID := "v25.10.19.1"

	// Delete the stage if it exists
	helpers.API_DeleteStage(t, app, payload.Token, releaseID+"-test")

	// Create a stage
	createBody, createStatusCode := helpers.API_CreateStage(t, app, payload.Token, releaseID, "test")
	require.Equal(t, http.StatusCreated, createStatusCode)

	var createPayload struct {
		Stage models.Stage `json:"stage"`
	}

	err = json.Unmarshal(createBody, &createPayload)
	require.NoError(t, err)
	require.NotEmpty(t, createPayload.Stage.ID)

	// List stages
	listBody, listStatusCode := helpers.API_ListStages(t, app, payload.Token)
	require.Equal(t, http.StatusOK, listStatusCode)

	var listPayload struct {
		Stages []models.Stage `json:"stages"`
	}

	err = json.Unmarshal(listBody, &listPayload)
	require.NoError(t, err)
	require.NotEmpty(t, listPayload.Stages)

	// Get the stage
	getBody, getStatusCode := helpers.API_GetStage(t, app, payload.Token, createPayload.Stage.ID)
	require.Equal(t, http.StatusOK, getStatusCode)

	var getPayload struct {
		Stage models.Stage `json:"stage"`
	}

	err = json.Unmarshal(getBody, &getPayload)
	require.NoError(t, err)
	require.Equal(t, createPayload.Stage.ID, getPayload.Stage.ID)

	// Start a test run

	startTestBody, startTestStatusCode := helpers.API_StartTestRun(t, app, payload.Token, createPayload.Stage.ID)

	require.Equal(t, http.StatusCreated, startTestStatusCode)

	var startTestPayload models.Test

	err = json.Unmarshal(startTestBody, &startTestPayload)

	require.NoError(t, err)

}
