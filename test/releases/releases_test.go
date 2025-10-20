package releases

import (
	"encoding/json"
	"net/http"
	"testing"

	"hypervisor/internal/models"
	"hypervisor/test/helpers"

	"github.com/stretchr/testify/require"
)

func TestListReleases(t *testing.T) {
	// Login to get a token
	body, statusCode := helpers.API_HyperUsersLogin(t, app, "testhyperuser", "testhyperuser")
	require.Equal(t, http.StatusOK, statusCode)

	var payload struct {
		Token string `json:"token"`
	}
	err := json.Unmarshal(body, &payload)
	require.NoError(t, err)

	// List releases
	body, statusCode = helpers.API_ListReleases(t, app, payload.Token)
	require.Equal(t, http.StatusOK, statusCode)

	var releasesPayload struct {
		Releases []models.Release `json:"releases"`
	}

	err = json.Unmarshal(body, &releasesPayload)
	require.NoError(t, err)

	if len(releasesPayload.Releases) == 0 {
		t.Log("no releases found, which might be okay if the database is empty")
	}
}
