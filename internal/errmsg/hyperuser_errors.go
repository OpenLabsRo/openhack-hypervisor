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
