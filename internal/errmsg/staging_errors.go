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
    StageSessionNotFound = NewStatusError(
        http.StatusNotFound,
        "stage session not found",
    )
    StageMissingEnv = NewStatusError(
        http.StatusConflict,
        "stage is missing an environment submission",
    )
    DeploymentInvalidRequest = NewStatusError(
        http.StatusBadRequest,
        "invalid deployment request payload",
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

type _StageSessionNotFound struct {
    StatusCode int    `json:"statusCode" example:"404"`
    Message    string `json:"message" example:"stage session not found"`
}

type _StageMissingEnv struct {
    StatusCode int    `json:"statusCode" example:"409"`
    Message    string `json:"message" example:"stage is missing an environment submission"`
}

type _DeploymentInvalidRequest struct {
    StatusCode int    `json:"statusCode" example:"400"`
    Message    string `json:"message" example:"invalid deployment request payload"`
}
