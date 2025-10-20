package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"hypervisor/internal/errmsg"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func RequestRunner(
	t *testing.T,
	app *fiber.App,
	method string,
	path string,
	sendBytes []byte,
	token *string,
	config ...fiber.TestConfig,
) (bodyBytes []byte, statusCode int) {
	config = append(config, fiber.TestConfig{Timeout: 300 * time.Second})
	req, err := http.NewRequest(
		method,
		path,
		bytes.NewBuffer(sendBytes),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	if token != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
	}

	// send request to the shared app
	var res *http.Response
	if len(config) > 0 {
		res, err = app.Test(req, config[0])
	} else {
		res, err = app.Test(req)
	}
	require.NoError(t, err)

	statusCode = res.StatusCode

	bodyBytes, err = io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err) // or handle error normally
	}
	defer res.Body.Close()

	return
}

func ResponseErrorCheck(
	t *testing.T,
	app *fiber.App,
	serr errmsg.StatusError,
	bodyBytes []byte,
	statusCode int,
) {
	require.Equal(t, serr.StatusCode, statusCode)

	var body struct {
		Message string `json:"message"`
	}
	err := json.Unmarshal(bodyBytes, &body)
	require.NoError(t, err)

	require.Equal(t, serr.Message, body.Message)
}
