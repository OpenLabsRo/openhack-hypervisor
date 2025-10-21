package proxy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"hypervisor/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
)

// RouteMap holds the current routing configuration
type RouteMap struct {
	mu          sync.RWMutex
	deployments map[string]*models.Deployment // stageID -> deployment
	mainID      string                        // ID of main deployment
}

// NewRouteMap creates a new route map
func NewRouteMap() *RouteMap {
	return &RouteMap{
		deployments: make(map[string]*models.Deployment),
	}
}

// UpdateDeployment updates or adds a deployment to the route map
func (rm *RouteMap) UpdateDeployment(dep *models.Deployment) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if dep.Status == models.DeploymentStatusReady {
		rm.deployments[dep.StageID] = dep
	} else {
		delete(rm.deployments, dep.StageID)
	}

	// Check if this is the main deployment
	if dep.PromotedAt != nil {
		rm.mainID = dep.ID
	} else if rm.mainID == dep.ID {
		// This was the main deployment but is no longer promoted
		rm.mainID = ""
	}

	// Print updated routing map for monitoring
	rm.printRoutingMap()
}

// RemoveDeployment removes a deployment from the route map
func (rm *RouteMap) RemoveDeployment(deploymentID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for stageID, dep := range rm.deployments {
		if dep.ID == deploymentID {
			delete(rm.deployments, stageID)
			break
		}
	}

	if rm.mainID == deploymentID {
		rm.mainID = ""
	}

	// Print updated routing map for monitoring
	rm.printRoutingMap()
}

// GetDeployment returns the deployment for a given stage ID
func (rm *RouteMap) GetDeployment(stageID string) (*models.Deployment, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	dep, exists := rm.deployments[stageID]
	return dep, exists
}

// GetMainDeployment returns the main deployment
func (rm *RouteMap) GetMainDeployment() (*models.Deployment, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rm.mainID == "" {
		return nil, false
	}

	for _, dep := range rm.deployments {
		if dep.ID == rm.mainID {
			return dep, true
		}
	}

	return nil, false
}

// printRoutingMap prints the current routing configuration for debugging/monitoring
func (rm *RouteMap) printRoutingMap() {
	fmt.Println("=== Routing Map ===")

	if rm.mainID != "" {
		if mainDep, exists := rm.deployments[rm.mainID]; exists && mainDep.Port != nil {
			fmt.Printf("Main (/): %s (stage: %s) -> localhost:%d\n", rm.mainID, mainDep.StageID, *mainDep.Port)
		} else {
			fmt.Printf("Main (/): %s (no port assigned)\n", rm.mainID)
		}
	} else {
		fmt.Println("Main (/): none")
	}

	if len(rm.deployments) > 0 {
		fmt.Println("Stages:")
		for stageID, dep := range rm.deployments {
			if dep.Port != nil {
				fmt.Printf("  /%s/* -> localhost:%d\n", stageID, *dep.Port)
			} else {
				fmt.Printf("  /%s/* -> no port assigned\n", stageID)
			}
		}
	} else {
		fmt.Println("Stages: none")
	}

	fmt.Println("==================")
}

// SetupRoutes sets up the proxy routes on the Fiber app
func (rm *RouteMap) SetupRoutes(app *fiber.App) {
	// Single middleware that handles all proxy routing
	app.Use("/*", func(c fiber.Ctx) error {
		path := c.Path()

		// Skip API routes
		if len(path) >= 11 && path[:11] == "/hypervisor" {
			return c.Next()
		}

		// Check if this looks like a stage request (has segments)
		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) > 0 && pathParts[0] != "" {
			stageID := pathParts[0]

			// Skip API routes that might match
			if stageID == "hypervisor" || stageID == "ws" {
				return c.Next()
			}

			// Check if this stage exists
			if dep, exists := rm.GetDeployment(stageID); exists && dep.Port != nil {
				target := fmt.Sprintf("http://localhost:%d", *dep.Port)

				// Remove stage prefix from path
				remainingPath := "/"
				stagePrefix := "/" + stageID
				if len(path) > len(stagePrefix) && path[len(stagePrefix)] == '/' {
					remainingPath = path[len(stagePrefix):]
				}

				finalURL := target + remainingPath
				return proxy.Forward(finalURL)(c)
			}
		}

		// Check for main deployment (root path)
		if mainDep, exists := rm.GetMainDeployment(); exists && mainDep.Port != nil {
			target := fmt.Sprintf("http://localhost:%d", *mainDep.Port)
			finalURL := target + path
			return proxy.Forward(finalURL)(c)
		}

		return c.Next()
	})
}

// LoadFromDatabase loads all current deployments from the database
func (rm *RouteMap) LoadFromDatabase(ctx context.Context) error {
	deployments, err := models.GetAllDeployments(ctx)
	if err != nil {
		return fmt.Errorf("failed to load deployments: %w", err)
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.deployments = make(map[string]*models.Deployment)
	rm.mainID = ""

	for _, dep := range deployments {
		dep := dep // copy
		if dep.Status == models.DeploymentStatusReady {
			rm.deployments[dep.StageID] = &dep
		}
		if dep.PromotedAt != nil {
			rm.mainID = dep.ID
		}
	}

	// Print initial routing map on startup
	rm.printRoutingMap()
	return nil
}

// StartWatcher starts a goroutine that watches for deployment changes
func (rm *RouteMap) StartWatcher(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Poll every 30 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := rm.LoadFromDatabase(ctx); err != nil {
					// Error updating route map - could add proper error handling here
				}
			}
		}
	}()
}

// Global route map instance
var GlobalRouteMap *RouteMap

// InitProxy initializes the global proxy system
func InitProxy(ctx context.Context) error {
	GlobalRouteMap = NewRouteMap()

	if err := GlobalRouteMap.LoadFromDatabase(ctx); err != nil {
		return err
	}

	// GlobalRouteMap.StartWatcher(ctx) // TODO: implement signal-based updates from deployments
	return nil
}
