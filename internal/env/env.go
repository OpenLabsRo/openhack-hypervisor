package env

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// actual environment variables
var JWT_SECRET []byte
var MONGO_URI string
var GITHUB_WEBHOOK_SECRET string
var PREFORK bool
var DRAIN_MODE bool

// this is required
var VERSION string

type Config struct {
	Root       string
	AppVersion string
}

func Init(envRoot string, appVersion string) {
	loadEnv(envRoot)
	loadVersion(appVersion)

	PREFORK, _ = strconv.ParseBool(os.Getenv("PREFORK"))
	DRAIN_MODE, _ = strconv.ParseBool(os.Getenv("DRAIN_MODE"))
	MONGO_URI = os.Getenv("MONGO_URI")
	JWT_SECRET = []byte(os.Getenv("JWT_SECRET"))
	GITHUB_WEBHOOK_SECRET = strings.TrimSpace(os.Getenv("GITHUB_WEBHOOK_SECRET"))
}

func loadEnv(envRoot string) {
	if envRoot == "" {
		envRoot = repoRoot()
	}

	path := path.Join(envRoot, ".env")
	if err := godotenv.Overload(path); err != nil {
		log.Fatalf("failed to load env file %s: %v", path, err)
	}
}

func loadVersion(appVersion string) {
	if appVersion == "" {
		data, err := os.ReadFile(filepath.Join(repoRoot(), "VERSION"))
		if err != nil {
			log.Fatalf("failed to read version file from repo root: %v", err)
		}

		trimmed := strings.TrimSpace(string(data))
		if trimmed != "" {
			VERSION = trimmed
		} else {
			VERSION = "unknown"
		}
	} else {
		VERSION = appVersion
	}
}

func repoRoot() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(b), "../..")
}
