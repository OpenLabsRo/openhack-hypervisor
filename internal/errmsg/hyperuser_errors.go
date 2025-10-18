package errmsg

import "net/http"

var (
	HyperUserNotExists = NewStatusError(
		http.StatusNotFound,
		"hyperuser does not exist",
	)
	HyperUserNoToken = NewStatusError(
		http.StatusUnauthorized,
		"no token has been provided",
	)
	HyperUserWrongPassword = NewStatusError(
		http.StatusUnauthorized,
		"username or password is incorrect",
	)
	HyperUserInvalidPayload = NewStatusError(
		http.StatusBadRequest,
		"username and password must be provided",
	)
)

type _HyperUserNotExists struct {
	StatusCode int    `json:"statusCode" example:"404"`
	Message    string `json:"message" example:"hyperuser does not exist"`
}

type _HyperUserNoToken struct {
	StatusCode int    `json:"statusCode" example:"401"`
	Message    string `json:"message" example:"no token has been provided"`
}

type _HyperUserWrongPassword struct {
	StatusCode int    `json:"statusCode" example:"401"`
	Message    string `json:"message" example:"username or password is incorrect"`
}

type _HyperUserInvalidPayload struct {
	StatusCode int    `json:"statusCode" example:"400"`
	Message    string `json:"message" example:"username and password must be provided"`
}
