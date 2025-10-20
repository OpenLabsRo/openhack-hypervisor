package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"hypervisor/internal/core"
	"hypervisor/internal/errmsg"
	"hypervisor/internal/events"
	"hypervisor/internal/fs"
	"hypervisor/internal/models"
	"hypervisor/internal/paths"
	"hypervisor/internal/proxy"
	"hypervisor/internal/systemd"
	"hypervisor/internal/utils"
	"hypervisor/internal/ws"

	"github.com/gofiber/fiber/v3"
)

// ListDeploymentsHandler lists all deployments.
// @Summary List deployments
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce json
// @Success 200 {array} models.Deployment
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments [get]
func ListDeploymentsHandler(c fiber.Ctx) error {
	deployments, err := models.GetAllDeployments(context.Background())
	if err != nil {
		return utils.StatusError(c, err)
	}

	return c.JSON(deployments)
}

// GetDeploymentHandler gets a single deployment.
// @Summary Get deployment
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Success 200 {object} models.Deployment
// @Failure 404 {object} errmsg._DeploymentNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments/{deploymentId} [get]
func GetDeploymentHandler(c fiber.Ctx) error {
	deploymentID := c.Params("deploymentId")
	dep, err := models.GetDeploymentByID(context.Background(), deploymentID)
	if err != nil {
		return utils.StatusError(c, errmsg.DeploymentNotFound)
	}

	return c.JSON(dep)
}

// PromoteDeploymentHandler promotes a deployment to main.
// @Summary Promote deployment to main
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Success 200 {object} models.Deployment
// @Failure 404 {object} errmsg._DeploymentNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments/{deploymentId}/promote [post]
func PromoteDeploymentHandler(c fiber.Ctx) error {
	deploymentID := c.Params("deploymentId")
	dep, err := models.GetDeploymentByID(context.Background(), deploymentID)
	if err != nil {
		return utils.StatusError(c, errmsg.DeploymentNotFound)
	}

	// For now, just update PromotedAt
	now := time.Now()
	dep.PromotedAt = &now
	if err := models.UpdateDeployment(context.Background(), *dep); err != nil {
		return utils.StatusError(c, err)
	}

	// Update proxy with promoted deployment
	proxy.GlobalRouteMap.UpdateDeployment(dep)

	// Note: main deployment is tracked via PromotedAt in database, no cache needed

	if events.Em != nil {
		events.Em.DeploymentPromoted(*dep)
	}

	return c.JSON(dep)
}

// ShutdownDeploymentHandler stops a deployment.
// @Summary Shutdown deployment
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Success 200 {object} models.Deployment
// @Failure 404 {object} errmsg._DeploymentNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments/{deploymentId}/shutdown [post]
func ShutdownDeploymentHandler(c fiber.Ctx) error {
	deploymentID := c.Params("deploymentId")
	dep, err := models.GetDeploymentByID(context.Background(), deploymentID)
	if err != nil {
		return utils.StatusError(c, errmsg.DeploymentNotFound)
	}

	if err := systemd.StopBackendService(deploymentID); err != nil {
		return utils.StatusError(c, err)
	}

	dep.Status = "stopped"
	dep.PromotedAt = nil // Clear promotion when shutting down
	if err := models.UpdateDeployment(context.Background(), *dep); err != nil {
		return utils.StatusError(c, err)
	}

	// Update proxy with stopped deployment
	proxy.GlobalRouteMap.UpdateDeployment(dep)

	if events.Em != nil {
		events.Em.DeploymentStopped(*dep)
	}

	return c.JSON(dep)
}

// DeleteDeploymentHandler deletes a deployment.
// @Summary Delete deployment
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Param force query bool false "Force stop before delete"
// @Success 204
// @Failure 404 {object} errmsg._DeploymentNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments/{deploymentId} [delete]
func DeleteDeploymentHandler(c fiber.Ctx) error {
	deploymentID := c.Params("deploymentId")
	dep, err := models.GetDeploymentByID(context.Background(), deploymentID)
	if err != nil {
		return utils.StatusError(c, errmsg.DeploymentNotFound)
	}

	force := c.Query("force") == "true"
	if force && dep.Status == "ready" {
		if err := systemd.StopBackendService(deploymentID); err != nil {
			return utils.StatusError(c, err)
		}
	}

	if err := systemd.DisableBackendService(deploymentID); err != nil {
		return utils.StatusError(c, err)
	}

	if err := systemd.RemoveBackendServiceFile(deploymentID); err != nil {
		return utils.StatusError(c, err)
	}

	// Remove from proxy before deleting from database
	proxy.GlobalRouteMap.RemoveDeployment(deploymentID)

	if err := models.DeleteDeployment(context.Background(), deploymentID); err != nil {
		return utils.StatusError(c, err)
	}

	// Delete the built binary
	versionWithoutV := strings.TrimPrefix(dep.Version, "v")
	binaryPath := filepath.Join(paths.OpenHackBuildsDir, versionWithoutV)
	if err := fs.Remove(binaryPath); err != nil {
		// Log error but don't fail the deletion
		fmt.Printf("Warning: failed to remove binary %s: %v\n", binaryPath, err)
	}

	// Reset stage status to ready for redeployment
	stage, err := models.GetStageByID(context.Background(), dep.StageID)
	if err == nil {
		stage.Status = models.StageStatusReady
		stage.UpdatedAt = time.Now()
		models.UpdateStage(context.Background(), *stage)
	}

	if events.Em != nil {
		events.Em.DeploymentDeleted(*dep)
	}

	return c.SendStatus(http.StatusNoContent)
}

// StreamDeploymentLogs upgrades the connection and streams deployment logs.
// @Summary Stream deployment logs
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Param deploymentId path string true "Deployment ID"
// @Router /hypervisor/ws/deployments/{deploymentId}/logs [get]
func StreamDeploymentLogs(c fiber.Ctx) error {
	deploymentID := c.Params("deploymentId")

	if deploymentID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing deployment identifier")
	}

	return ws.StreamWebSocket(c, func(ctx context.Context, writer *ws.WebsocketLogWriter) error {
		dep, err := models.GetDeploymentByID(ctx, deploymentID)
		if err != nil {
			writer.WriteStatus("error", fmt.Sprintf("deployment not found: %v", err))
			return err
		}

		if dep.Status == "ready" {
			// Stream runtime logs from journalctl
			return streamJournalctl(ctx, deploymentID, writer)
		} else {
			// Stream provisioning logs from file
			if dep.LogPath == "" {
				writer.WriteStatus("error", "log path is not available for this deployment")
				return errors.New("log path not available")
			}
			return core.StreamDeploymentLogFile(ctx, dep.LogPath, deploymentID, writer, writer)
		}
	})
}

func streamJournalctl(ctx context.Context, deploymentID string, writer *ws.WebsocketLogWriter) error {
	cmd := exec.CommandContext(ctx, "journalctl", "-u", systemd.ServiceName(deploymentID), "-f", "-n", "100")
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}

// CreateDeploymentHandler creates a new deployment by promoting a stage.
// @Summary Create deployment by promoting a stage
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce json
// @Param stageId path string true "Stage ID to promote"
// @Success 201 {object} models.Deployment
// @Failure 400 {object} errmsg._DeploymentInvalidRequest
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments/{stageId} [post]
func CreateDeploymentHandler(c fiber.Ctx) error {
	stageID := c.Params("stageId")
	if stageID == "" {
		return utils.StatusError(c, errmsg.DeploymentInvalidRequest)
	}

	// Check if deployment already exists for this stage
	if existing, err := models.GetDeploymentByID(context.Background(), stageID); err == nil {
		// Deployment exists, redeploy it
		go core.ProvisionDeployment(*existing)
		// Update proxy with existing deployment
		proxy.GlobalRouteMap.UpdateDeployment(existing)
		return c.Status(http.StatusOK).JSON(existing)
	} else if err != mongo.ErrNoDocuments {
		// Some other error
		return utils.StatusError(c, err)
	}

	// No existing deployment, create new one
	deployment, err := core.PromoteStage(context.Background(), stageID)
	if err != nil {
		return utils.StatusError(c, err)
	}

	// Update proxy with new deployment
	proxy.GlobalRouteMap.UpdateDeployment(deployment)

	if events.Em != nil {
		events.Em.DeploymentCreated(*deployment)
	}

	return c.Status(http.StatusCreated).JSON(deployment)
}

// GetMainRouteHandler gets the current main deployment.
// @Summary Get main route
// @Tags Hypervisor Routes
// @Security HyperUserAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/routes/main [get]
func GetMainRouteHandler(c fiber.Ctx) error {
	if mainDep, exists := proxy.GlobalRouteMap.GetMainDeployment(); exists {
		return c.JSON(fiber.Map{"deploymentId": mainDep.ID})
	}

	return c.JSON(fiber.Map{"deploymentId": "none"})
}

// SetMainRouteHandler sets the main deployment.
// @Summary Set main route
// @Tags Hypervisor Routes
// @Security HyperUserAuth
// @Accept json
// @Produce json
// @Param payload body map[string]string true "Route payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errmsg._DeploymentInvalidRequest
// @Failure 404 {object} errmsg._DeploymentNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/routes/main [put]
func SetMainRouteHandler(c fiber.Ctx) error {
	var req map[string]string
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return utils.StatusError(c, errmsg.DeploymentInvalidRequest)
	}

	deploymentID, ok := req["deploymentId"]
	if !ok || deploymentID == "" {
		return utils.StatusError(c, errmsg.DeploymentInvalidRequest)
	}

	// Validate deployment exists
	dep, err := models.GetDeploymentByID(context.Background(), deploymentID)
	if err != nil {
		return utils.StatusError(c, errmsg.DeploymentNotFound)
	}

	// Clear PromotedAt from current main deployment if any
	if currentMain, exists := proxy.GlobalRouteMap.GetMainDeployment(); exists && currentMain.ID != deploymentID {
		currentMain.PromotedAt = nil
		if err := models.UpdateDeployment(context.Background(), *currentMain); err != nil {
			return utils.StatusError(c, err)
		}
		proxy.GlobalRouteMap.UpdateDeployment(currentMain)
	}

	// Set PromotedAt on new main deployment
	now := time.Now()
	dep.PromotedAt = &now
	if err := models.UpdateDeployment(context.Background(), *dep); err != nil {
		return utils.StatusError(c, err)
	}

	// Update proxy with new main deployment
	proxy.GlobalRouteMap.UpdateDeployment(dep)

	return c.JSON(fiber.Map{"deploymentId": dep.ID})
}
