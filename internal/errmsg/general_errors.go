package errmsg

import "net/http"

func InternalServerError(err error) StatusError {
    return NewStatusError(
        http.StatusInternalServerError,
        "internal server error: "+err.Error(),
    )
}
