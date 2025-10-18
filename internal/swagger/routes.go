package swagger

import (
	"encoding/json"
	"errors"
	"fmt"
	"hypervisor/internal/env"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	embedJSONPath = "docs/swagger.json"
	diskJSONPath  = "internal/swagger/docs/swagger.json"
	swaggerUIPath = "https://openhack-swagger.vercel.app/"
)

var uiTemplate = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>OpenHack Hypervisor API Docs</title>
  <link rel="icon" type="image/png" href="https://dl.openhack.ro/icons/logo.png" />
  <link rel="stylesheet" href="%s/swagger-ui.css" />
  <style>
html, body { margin: 0; padding: 0; background: #000000; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="%s/swagger-ui-bundle.js"></script>
  <script src="%s/swagger-ui-standalone-preset.js"></script>
  <script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
      url: '/hypervisor/docs/doc.json',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      layout: 'StandaloneLayout',
      deepLinking: true,
      displayRequestDuration: true,
      persistAuthorization: true,
      requestInterceptor: (req) => {
        const authHeader = req.headers && req.headers.Authorization;
        if (authHeader && !/^Bearer /i.test(authHeader)) {
          req.headers.Authorization = 'Bearer ' + authHeader;
        }
        return req;
      },
    });
  };
  </script>
</body>
</html>`, swaggerUIPath, swaggerUIPath, swaggerUIPath)

// Register wires swagger-ui routes backed by the generated doc files.
func Register(router fiber.Router) {
	if router == nil {
		return
	}

	router.Get("/hypervisor/docs", func(c fiber.Ctx) error {
		c.Type("html", "utf-8")
		return c.SendString(uiTemplate)
	})

	router.Get("/hypervisor/docs/doc.json", func(c fiber.Ctx) error {
		data, err := loadDoc(embedJSONPath, diskJSONPath)
		if err != nil {
			return missingSpec(c, err)
		}

		data = applyDocDefaults(data)

		c.Type("json", "utf-8")
		return c.Send(data)
	})
}

func missingSpec(c fiber.Ctx, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Swagger spec not generated yet. Run `swag init -g cmd/server/main.go -o internal/swagger/docs` to create doc.json.",
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"message": "Failed to read swagger spec",
		"error":   err.Error(),
		"path":    filepath.Clean(diskJSONPath),
	})
}

func loadDoc(embedPath string, diskPath string) ([]byte, error) {
	data, err := swaggerDocs.ReadFile(embedPath)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	data, err = os.ReadFile(diskPath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func applyDocDefaults(data []byte) []byte {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return data
	}

	info := ensureMap(doc, "info")
	if version := strings.TrimSpace(env.VERSION); version != "" {
		info["version"] = version
	}

	doc["basePath"] = "/"

	encoded, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return data
	}

	if len(encoded) == 0 || encoded[len(encoded)-1] != '\n' {
		encoded = append(encoded, '\n')
	}

	return encoded
}

func ensureMap(root map[string]any, key string) map[string]any {
	val, ok := root[key]
	if ok {
		if existing, ok := val.(map[string]any); ok {
			return existing
		}
	}

	created := map[string]any{}
	root[key] = created
	return created
}
