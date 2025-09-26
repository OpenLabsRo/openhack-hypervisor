package systemd

import "embed"

//go:embed openhack-hypervisor.service
var HypervisorService embed.FS

const HypervisorServiceName = "openhack-hypervisor.service"
