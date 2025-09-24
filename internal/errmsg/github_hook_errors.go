package errmsg

import "net/http"

// GitHub webhook specific StatusError helpers surfaced by the handler.
var (
	GitHubSecretNotConfigured = NewStatusError(http.StatusInternalServerError, "webhook secret not configured")
	GitHubSignatureMissing    = NewStatusError(http.StatusBadRequest, "missing X-Hub-Signature-256 header")
	GitHubSignatureInvalid    = NewStatusError(http.StatusUnauthorized, "invalid webhook signature")
	GitHubEventMissing        = NewStatusError(http.StatusBadRequest, "missing X-GitHub-Event header")
	GitHubDeliveryMissing     = NewStatusError(http.StatusBadRequest, "missing X-GitHub-Delivery header")
	GitHubInvalidPayload      = NewStatusError(http.StatusBadRequest, "invalid webhook payload")
)
