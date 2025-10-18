package errmsg

var EmptyStatusError = NewStatusError(0, "")

type StatusError struct {
	StatusCode int
	Message    string
}

func NewStatusError(statusCode int, message string) StatusError {
	return StatusError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func (se StatusError) Error() string {
	return se.Message
}
