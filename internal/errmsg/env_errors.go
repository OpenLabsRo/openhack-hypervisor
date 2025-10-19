package errmsg

import "net/http"

var (
	EnvTemplateInvalidRequest = NewStatusError(
		http.StatusBadRequest,
		"invalid template env payload",
	)
)

type _EnvTemplateInvalidRequest struct {
	StatusCode int    `json:"statusCode" example:"400"`
	Message    string `json:"message" example:"invalid template env payload"`
}
