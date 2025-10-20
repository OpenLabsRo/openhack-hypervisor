package systemd

import "embed"

//go:embed openhack-backend.service
var BackendService embed.FS

const BackendServiceName = "openhack-backend.service"
