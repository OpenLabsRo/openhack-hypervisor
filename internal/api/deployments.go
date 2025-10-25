package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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

// GetRoutingMapHandler returns the current routing map.
// @Summary Get routing map
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce plain
// @Success 200 {string} string "Routing map as formatted text"
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/routing [get]
func GetRoutingMapHandler(c fiber.Ctx) error {
	if proxy.GlobalRouteMap == nil {
		return utils.StatusError(c, fmt.Errorf("routing map not initialized"))
	}

	routingMap := proxy.GlobalRouteMap.GetRoutingMap()
	return c.SendString(routingMap)
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

	dep.Status = models.DeploymentStatusStopped
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

// StartDeploymentHandler starts a stopped deployment.
// @Summary Start deployment
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Success 200 {object} models.Deployment
// @Failure 404 {object} errmsg._DeploymentNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments/{deploymentId}/start [post]
func StartDeploymentHandler(c fiber.Ctx) error {
	deploymentID := c.Params("deploymentId")
	dep, err := models.GetDeploymentByID(context.Background(), deploymentID)
	if err != nil {
		return utils.StatusError(c, errmsg.DeploymentNotFound)
	}

	if err := systemd.StartBackendService(deploymentID); err != nil {
		return utils.StatusError(c, err)
	}

	dep.Status = models.DeploymentStatusReady
	if err := models.UpdateDeployment(context.Background(), *dep); err != nil {
		return utils.StatusError(c, err)
	}

	// Update proxy with restarted deployment
	proxy.GlobalRouteMap.UpdateDeployment(dep)

	if events.Em != nil {
		events.Em.DeploymentCreated(*dep)
	}

	return c.JSON(dep)
}

// DeleteDeploymentHandler deletes a deployment.
// @Summary Delete deployment
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Param force query bool false "Force stop before delete, or force delete main deployment"
// @Success 204
// @Failure 404 {object} errmsg._DeploymentNotFound
// @Failure 409 {object} errmsg._CannotDeleteMainDeployment
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments/{deploymentId} [delete]
func DeleteDeploymentHandler(c fiber.Ctx) error {
	deploymentID := c.Params("deploymentId")
	dep, err := models.GetDeploymentByID(context.Background(), deploymentID)
	if err != nil {
		return utils.StatusError(c, errmsg.DeploymentNotFound)
	}

	force := c.Query("force") == "true"

	// Prevent deleting main deployment unless force=true
	if dep.PromotedAt != nil && !force {
		return utils.StatusError(c, errmsg.CannotDeleteMainDeployment)
	}

	if dep.Status == models.DeploymentStatusReady {
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
	log.Printf("DEBUG: StreamDeploymentLogs called for deploymentID=%s", deploymentID)

	if deploymentID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing deployment identifier")
	}

	return ws.StreamWebSocket(c, func(ctx context.Context, writer *ws.WebsocketLogWriter) error {
		log.Printf("DEBUG: WebSocket connected for deploymentID=%s", deploymentID)

		dep, err := models.GetDeploymentByID(ctx, deploymentID)
		if err != nil {
			errMsg := fmt.Sprintf("deployment not found: %v", err)
			log.Printf("DEBUG: %s", errMsg)
			writer.WriteStatus("error", errMsg)
			return err
		}

		log.Printf("DEBUG: Deployment found with status=%s", dep.Status)

		// Stream provisioning logs from file
		log.Printf("DEBUG: Streaming deployment logs from file logPath=%s", dep.LogPath)
		if dep.LogPath == "" {
			writer.WriteStatus("error", "log path is not available for this deployment")
			return errors.New("log path not available")
		}
		return core.StreamDeploymentLogFile(ctx, dep.LogPath, deploymentID, writer, writer)
	})
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
		// Deployment exists. Always perform full redeploy: stop and remove existing service, then create new deployment
		if existing.Status == models.DeploymentStatusReady {
			if err := systemd.StopBackendService(existing.ID); err != nil {
				return utils.StatusError(c, err)
			}
		}

		if err := systemd.DisableBackendService(existing.ID); err != nil {
			return utils.StatusError(c, err)
		}

		if err := systemd.RemoveBackendServiceFile(existing.ID); err != nil {
			return utils.StatusError(c, err)
		}

		// Remove from proxy before deleting from database
		proxy.GlobalRouteMap.RemoveDeployment(existing.ID)

		if err := models.DeleteDeployment(context.Background(), existing.ID); err != nil {
			return utils.StatusError(c, err)
		}

		// Delete the built binary (best-effort)
		versionWithoutV := strings.TrimPrefix(existing.Version, "v")
		binaryPath := filepath.Join(paths.OpenHackBuildsDir, versionWithoutV)
		if err := fs.Remove(binaryPath); err != nil {
			// Log warning but continue
			fmt.Printf("Warning: failed to remove binary %s: %v\n", binaryPath, err)
		}

		// Reset stage status to ready for redeployment
		stage, err := models.GetStageByID(context.Background(), existing.StageID)
		if err == nil {
			stage.Status = models.StageStatusReady
			stage.UpdatedAt = time.Now()
			models.UpdateStage(context.Background(), *stage)
		}

		if events.Em != nil {
			events.Em.DeploymentDeleted(*existing)
		}

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

	return c.Status(http.StatusCreated).JSON(bson.M{
		"deployment": deployment,
		"stageID":    stageID,
	})
}
