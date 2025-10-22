package nginx

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed nginx.conf
var nginxConf embed.FS

const (
	// DefaultNginxConfigPath is the path where the nginx config will be written
	// We use conf.d/app.conf which is the common location for drop-in site configs.
	DefaultNginxConfigPath = "/etc/nginx/conf.d/app.conf"
)

// Config represents the nginx configuration for the hypervisor
type Config struct {
	SSL          bool
	SSLCertPath  string
	SSLKeyPath   string
	UpstreamPort string
	Deployments  []DeploymentConfig
}

// DeploymentConfig represents a single deployment configuration
type DeploymentConfig struct {
	Name   string
	Domain string
	Port   string
	SSL    bool
	Cert   string
	Key    string
}

// Deployment represents a deployment from the hypervisor API
type APIDeployment struct {
	ID         string  `json:"id"`
	StageID    string  `json:"stageId"`
	Version    string  `json:"version"`
	EnvTag     string  `json:"envTag"`
	Port       *int    `json:"port,omitempty"`
	Status     string  `json:"status"`
	LogPath    string  `json:"logPath,omitempty"`
	PromotedAt *string `json:"promotedAt,omitempty"`
}

// GenerateConfig generates nginx configuration from the current hypervisor state
func GenerateConfig() (*Config, error) {
	// Get hypervisor host
	host := os.Getenv("HYPERVISOR_HOST")
	if host == "" {
		host = "localhost:8080"
	}

	// Fetch deployments from hypervisor API
	url := fmt.Sprintf("http://%s/hypervisor/deployments", host)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to hypervisor API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hypervisor API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var deployments []APIDeployment
	if err := json.Unmarshal(body, &deployments); err != nil {
		return nil, fmt.Errorf("failed to parse deployments: %w", err)
	}

	config := &Config{
		SSL:          false,
		UpstreamPort: "8080",
		Deployments:  []DeploymentConfig{},
	}

	// Process deployments
	for _, dep := range deployments {
		if dep.Status != "ready" || dep.Port == nil {
			continue // Skip non-ready deployments or those without ports
		}

		deploymentConfig := DeploymentConfig{
			Name:   dep.ID,
			Domain: fmt.Sprintf("%s.hypervisor.local", dep.StageID),
			Port:   fmt.Sprintf("%d", *dep.Port),
			SSL:    false,
		}

		config.Deployments = append(config.Deployments, deploymentConfig)
	}

	return config, nil
}

// WriteConfig writes the nginx configuration to a file
func WriteConfig(config *Config, outputPath string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate nginx config content
	content, err := renderTemplate(config)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// InstallConfig installs the nginx configuration and enables it
func InstallConfig(configPath string) error {
	// Read embedded nginx.conf and write to sites-available using sudo if necessary
	data, err := nginxConf.ReadFile("nginx.conf")
	if err != nil {
		return fmt.Errorf("failed to read embedded nginx.conf: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(DefaultNginxConfigPath), 0o755); err != nil {
		return fmt.Errorf("failed to create nginx config directory: %w", err)
	}

	if err := os.WriteFile(DefaultNginxConfigPath, data, 0o644); err != nil {
		// fallback to sudo tee if permission denied
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to write nginx config: %w", err)
		}
		cmd := exec.Command("sudo", "tee", DefaultNginxConfigPath)
		cmd.Stdin = bytes.NewReader(data)
		cmd.Stdout = io.Discard
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("sudo tee failed: %w", err)
		}
	}

	// For conf.d we don't need a symlink; nginx will pick up files in conf.d

	return nil
}

// EmbeddedConfig returns the raw embedded nginx.conf contents.
func EmbeddedConfig() (string, error) {
	data, err := nginxConf.ReadFile("nginx.conf")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// copyFile copies a file from src to dst
// copyFile removed: we now use embedded nginx.conf

// renderTemplate renders the nginx configuration template
func renderTemplate(config *Config) (string, error) {
	tmpl := `# /etc/nginx/conf.d/hypervisor.conf
# Blue = primary (8080), Green = standby (8081).
# When Blue restarts (or /hypervisor/meta/ping returns 503 during drain), traffic falls to Green.
# When Blue is healthy again, traffic snaps back without an nginx reload.

upstream blue {
    server 127.0.0.1:8080 max_fails=1 fail_timeout=2s max_conns=512;
    keepalive 64;
}
upstream green {
    server 127.0.0.1:8081 max_fails=1 fail_timeout=2s max_conns=512;
    keepalive 64;
}

# Upgrade header helper for WebSockets
map $http_upgrade $upgrade_hdr {
    default upgrade;
    ""      close;
}

server {
    listen 80{{if .SSL}} ssl{{end}};

    {{if .SSL}}
    ssl_certificate {{.SSLCertPath}};
    ssl_certificate_key {{.SSLKeyPath}};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    {{end}}

    # Fail fast so we flip to green quickly if blue is down/draining
    proxy_connect_timeout 0.5s;
    proxy_read_timeout    60s;   # long enough for idle WS pings
    proxy_send_timeout    30s;
    proxy_http_version    1.1;

    # Retry only on true failures/timeouts/5xx
    proxy_next_upstream error timeout http_502 http_503 http_504;
    proxy_next_upstream_timeout 2s;
    proxy_next_upstream_tries 2;

    # Common headers
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Connection "";  # keepalive for HTTP
    proxy_buffering off;             # good default for APIs/WS

    # --- Primary route: try BLUE, on 502/503/504 jump to GREEN
    location / {
        proxy_pass http://blue;
        error_page 502 503 504 = @green;
    }

    location @green {
        proxy_pass http://green;
    }

    # --- WebSocket endpoints: Use same fallback behavior
    location /hypervisor/ws/ {
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $upgrade_hdr;
        proxy_pass http://blue;
        error_page 502 503 504 = @green_ws;
    }

    location @green_ws {
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $upgrade_hdr;
        proxy_pass http://green;
    }

    # --- Health: Blue-first; if Blue is draining (503) or down, route to Green
    location /hypervisor/meta/ping {
        proxy_buffering off;
        proxy_pass http://blue;
        error_page 502 503 504 = @green_health;
    }

    location @green_health {
        proxy_buffering off;
        proxy_pass http://green;
    }
}

{{range .Deployments}}
server {
    listen 80{{if .SSL}} ssl{{end}};
    server_name {{.Domain}};

    {{if .SSL}}
    ssl_certificate {{.Cert}};
    ssl_certificate_key {{.Key}};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    {{end}}

    location / {
        proxy_pass http://localhost:{{.Port}};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
{{end}}`

	t := template.Must(template.New("nginx").Parse(tmpl))

	var buf strings.Builder
	if err := t.Execute(&buf, config); err != nil {
		return "", err
	}

	return buf.String(), nil
}
