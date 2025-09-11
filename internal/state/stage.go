package state

import "hypervisor/internal/db"

var (
	servingPort   string
	candidatePort string
)

func Init() {
	servingPort, _ = db.CacheGet("servingport")
	if servingPort == "" {
		servingPort = "9000"
	}
	candidatePort, _ = db.CacheGet("servingport")
	if candidatePort == "" {
		candidatePort = "9001"
	}
}

func SwitchPorts() (err error) {
	tempPort := servingPort

	// switching the serving first
	servingPort = candidatePort
	err = db.CacheSet("servingport", candidatePort)
	if err != nil {
		return
	}

	// and then the candidate
	candidatePort = tempPort
	err = db.CacheSet("candidateport", tempPort)
	if err != nil {
		return
	}

	return nil
}
