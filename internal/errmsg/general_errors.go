package errmsg

import "net/http"

func InternalServerError(err error) StatusError {
	return NewStatusError(
		http.StatusInternalServerError,
		"internal server error: "+err.Error(),
	)
}

type _InternalServerError struct {
	StatusCode int    `json:"statusCode" example:"500"`
	Message    string `json:"message" example:"internal server error: <details>"`
}
