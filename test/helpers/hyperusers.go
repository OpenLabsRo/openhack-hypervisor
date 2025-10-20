package helpers

import (
	"encoding/json"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func API_HyperUsersLogin(
	t *testing.T,
	app *fiber.App,
	username string,
	password string,
) (bodyBytes []byte, statusCode int) {
	// payload for login request
	payload := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	// marshalling the payload into JSON
	sendBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	return RequestRunner(t, app,
		"POST",
		"/hypervisor/hyperusers/login",
		sendBytes,
		nil,
	)
}

func API_HyperUsersPing(
	t *testing.T,
	app *fiber.App,
) (bodyBytes []byte, statusCode int) {
	return RequestRunner(t, app,
		"GET",
		"/hypervisor/hyperusers/ping",
		nil,
		nil,
	)
}
