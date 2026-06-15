package hyperusers

import (
	"encoding/json"
	"net/http"
	"testing"

	"hypervisor/test/helpers"

	"hypervisor/internal/errmsg"

	"github.com/stretchr/testify/require"
)

const (
	testHyperUserUsername = "testhyperuser"
	testHyperUserPassword = "testhyperuser"
)

func TestHyperUsersPing(t *testing.T) {
	body, statusCode := helpers.API_HyperUsersPing(t, app)
	require.Equal(t, http.StatusOK, statusCode)
	require.Equal(t, "PONG", string(body))
}

func TestHyperUsersLoginSuccess(t *testing.T) {
	body, statusCode := helpers.API_HyperUsersLogin(t, app, testHyperUserUsername, testHyperUserPassword)
	require.Equal(t, http.StatusOK, statusCode)

	var payload struct {
		Token     string `json:"token"`
		HyperUser struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"hyperuser"`
	}

	err := json.Unmarshal(body, &payload)
	require.NoError(t, err)

	require.NotEmpty(t, payload.Token)
	require.Equal(t, testHyperUserUsername, payload.HyperUser.Username)
	require.Empty(t, payload.HyperUser.Password)
}

func TestHyperUsersLoginWrongPassword(t *testing.T) {
	body, statusCode := helpers.API_HyperUsersLogin(t, app, testHyperUserUsername, "wrong-password")
	helpers.ResponseErrorCheck(t, app, errmsg.HyperUserWrongPassword, body, statusCode)
}

func TestHyperUsersLoginUserNotFound(t *testing.T) {
	body, statusCode := helpers.API_HyperUsersLogin(t, app, "missing-user", "whatever")
	helpers.ResponseErrorCheck(t, app, errmsg.HyperUserNotExists, body, statusCode)
}

func TestHyperUsersLoginInvalidPayload(t *testing.T) {
	body, statusCode := helpers.API_HyperUsersLogin(t, app, "", "")
	helpers.ResponseErrorCheck(t, app, errmsg.HyperUserInvalidPayload, body, statusCode)
}
