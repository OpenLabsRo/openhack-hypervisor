package errmsg

import "net/http"

var (
	StageInvalidRequest = NewStatusError(
		http.StatusBadRequest,
		"invalid stage request payload",
	)
	StageAlreadyExists = NewStatusError(
		http.StatusConflict,
		"stage already exists",
	)
	StageNotFound = NewStatusError(
		http.StatusNotFound,
		"stage not found",
	)
	StageReleaseNotFound = NewStatusError(
		http.StatusNotFound,
		"release not found",
	)
	StageMissingEnv = NewStatusError(
		http.StatusConflict,
		"stage is missing an environment update",
	)
	DeploymentAlreadyExists = NewStatusError(
		http.StatusConflict,
		"deployment already exists",
	)
	DeploymentInvalidRequest = NewStatusError(
		http.StatusBadRequest,
		"invalid deployment request payload",
	)
	DeploymentNotFound = NewStatusError(
		http.StatusNotFound,
		"deployment not found",
	)
	CannotDeleteMainDeployment = NewStatusError(
		http.StatusConflict,
		"cannot delete deployment marked as main - demote it first or use force=true",
	)
	NoDeploymentFound = NewStatusError(
		http.StatusNotFound,
		"no deployment found for this request - check that a deployment exists and is promoted to main",
	)
)

type _StageInvalidRequest struct {
	StatusCode int    `json:"statusCode" example:"400"`
	Message    string `json:"message" example:"invalid stage request payload"`
}

type _StageAlreadyExists struct {
	StatusCode int    `json:"statusCode" example:"409"`
	Message    string `json:"message" example:"stage already exists"`
}

type _StageNotFound struct {
	StatusCode int    `json:"statusCode" example:"404"`
	Message    string `json:"message" example:"stage not found"`
}

type _StageReleaseNotFound struct {
	StatusCode int    `json:"statusCode" example:"404"`
	Message    string `json:"message" example:"release not found"`
}

type _StageMissingEnv struct {
	StatusCode int    `json:"statusCode" example:"409"`
	Message    string `json:"message" example:"stage is missing an environment update"`
}

type _DeploymentInvalidRequest struct {
	StatusCode int    `json:"statusCode" example:"400"`
	Message    string `json:"message" example:"invalid deployment request payload"`
}

type _DeploymentNotFound struct {
	StatusCode int    `json:"statusCode" example:"404"`
	Message    string `json:"message" example:"deployment not found"`
}

type _CannotDeleteMainDeployment struct {
	StatusCode int    `json:"statusCode" example:"409"`
	Message    string `json:"message" example:"cannot delete deployment marked as main - demote it first or use force=true"`
}

type _NoDeploymentFound struct {
	StatusCode int    `json:"statusCode" example:"404"`
	Message    string `json:"message" example:"no deployment found for this request - check that a deployment exists and is promoted to main"`
}
