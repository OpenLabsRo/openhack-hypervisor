package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"hypervisor/internal/core"
	"hypervisor/internal/errmsg"
	"hypervisor/internal/models"
	"hypervisor/internal/utils"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/mongo"
)

type createStageRequest struct {
	ReleaseID string `json:"releaseId"`
	EnvTag    string `json:"envTag"`
	EnvText   string `json:"envText,omitempty"` // Optional - if provided, stage will be marked as ready
}

type StageResponse struct {
	Stage   models.Stage `json:"stage"`
	EnvText string       `json:"envText"`
}

type StageEnvResponse struct {
	EnvText string `json:"envText"`
}

type UpdateStageEnvRequest struct {
	EnvText *string `json:"envText"`
}

// CreateStageHandler prepares a new stage and returns the seeded template.
// @Summary Prepare stage
// @Description Bootstraps a stage by cloning the release repo and seeding the template environment.
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Accept json
// @Produce json
// @Param payload body createStageRequest true "Stage details"
// @Success 201 {object} StageResponse
// @Failure 400 {object} errmsg._StageInvalidRequest
// @Failure 404 {object} errmsg._StageReleaseNotFound
// @Failure 409 {object} errmsg._StageAlreadyExists
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages [post]
func CreateStageHandler(c fiber.Ctx) error {
	var req createStageRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	req.ReleaseID = strings.TrimSpace(req.ReleaseID)
	req.EnvTag = strings.TrimSpace(req.EnvTag)
	if req.ReleaseID == "" || req.EnvTag == "" {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	stage, _, err := core.PrepareStage(context.Background(), req.ReleaseID, req.EnvTag)
	if err != nil {
		return utils.StatusError(c, err)
	}

	envText := req.EnvText
	if envText == "" {
		// If no envText provided, read the template
		var readErr error
		envText, readErr = core.ReadStageEnv(stage.ID)
		if readErr != nil {
			return utils.StatusError(c, errmsg.InternalServerError(readErr))
		}
	} else {
		// If envText was provided, update the stage with it (marks as ready)
		updatedStage, err := core.UpdateStageEnv(context.Background(), stage.ID, envText)
		if err != nil {
			return utils.StatusError(c, err)
		}
		stage = updatedStage
	}

	return c.Status(http.StatusCreated).JSON(StageResponse{Stage: *stage, EnvText: envText})
}

type listStagesResponse struct {
	Stages []models.Stage `json:"stages"`
}

// ListStagesHandler returns all stages ordered by creation time.
// @Summary List stages
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Produce json
// @Success 200 {object} listStagesResponse
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages [get]
func ListStagesHandler(c fiber.Ctx) error {
	stages, err := models.ListStages(context.Background())
	if err != nil {
		return utils.StatusError(c, errmsg.InternalServerError(err))
	}

	return c.JSON(listStagesResponse{Stages: stages})
}

// GetStageHandler returns a single stage by id.
// @Summary Get stage
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Produce json
// @Param stageId path string true "Stage identifier"
// @Success 200 {object} StageResponse
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId} [get]
func GetStageHandler(c fiber.Ctx) error {
	stageID := strings.TrimSpace(c.Params("stageId"))
	if stageID == "" {
		return utils.StatusError(c, errmsg.StageNotFound)
	}

	stage, err := models.GetStageByID(context.Background(), stageID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return utils.StatusError(c, errmsg.StageNotFound)
		}
		return utils.StatusError(c, err)
	}

	envText, err := core.ReadStageEnv(stage.ID)
	if err != nil {
		return utils.StatusError(c, errmsg.InternalServerError(err))
	}

	return c.JSON(StageResponse{Stage: *stage, EnvText: envText})
}

// GetStageEnvHandler returns the current .env contents for a stage.
// @Summary Get stage env
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Produce json
// @Param stageId path string true "Stage identifier"
// @Success 200 {object} StageEnvResponse
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId}/env [get]
func GetStageEnvHandler(c fiber.Ctx) error {
	stageID := strings.TrimSpace(c.Params("stageId"))
	if stageID == "" {
		return utils.StatusError(c, errmsg.StageNotFound)
	}

	envText, err := core.ReadStageEnv(stageID)
	if err != nil {
		// If the file doesn't exist, it's effectively a 404.
		if os.IsNotExist(err) {
			return utils.StatusError(c, errmsg.StageNotFound)
		}
		return utils.StatusError(c, errmsg.InternalServerError(err))
	}

	return c.JSON(StageEnvResponse{EnvText: envText})
}

// UpdateStageEnvHandler writes new environment contents for a stage.
// @Summary Update stage env
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Accept json
// @Produce json
// @Param stageId path string true "Stage identifier"
// @Param payload body UpdateStageEnvRequest true "Env payload"
// @Success 200 {object} StageResponse
// @Failure 400 {object} errmsg._StageInvalidRequest
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId}/env [put]
func UpdateStageEnvHandler(c fiber.Ctx) error {
	stageID := strings.TrimSpace(c.Params("stageId"))
	if stageID == "" {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	var req UpdateStageEnvRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	if req.EnvText == nil {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	stage, err := core.UpdateStageEnv(context.Background(), stageID, *req.EnvText)
	if err != nil {
		return utils.StatusError(c, err)
	}

	return c.JSON(StageResponse{Stage: *stage, EnvText: *req.EnvText})
}

// DeleteStageHandler removes a stage and all associated resources.
// @Summary Delete stage
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Produce json
// @Param stageId path string true "Stage identifier"
// @Success 204
// @Failure 400 {object} errmsg._StageInvalidRequest
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId} [delete]
func DeleteStageHandler(c fiber.Ctx) error {
	stageID := strings.TrimSpace(c.Params("stageId"))
	if stageID == "" {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	if err := core.DeleteStage(context.Background(), stageID); err != nil {
		return utils.StatusError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}
