package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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
}

// CreateStageHandler prepares a new stage and returns the seeded template.
// @Summary Prepare stage
// @Description Bootstraps a stage by cloning the release repo and seeding the template environment.
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Accept json
// @Produce json
// @Param payload body createStageRequest true "Stage details"
// @Success 201 {object} models.Stage
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

	return c.Status(http.StatusCreated).JSON(stage)
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
// @Success 200 {object} models.Stage
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

	return c.JSON(stage)
}

type createStageSessionRequest struct {
	EnvText string `json:"envText"`
	Author  string `json:"author"`
	Notes   string `json:"notes"`
	Source  string `json:"source"`
}

type stageSessionResponse struct {
	Stage   models.Stage        `json:"stage"`
	Session models.StageSession `json:"session"`
}

// CreateStageSessionHandler records a new environment snapshot for a stage.
// @Summary Submit stage session
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Accept json
// @Produce json
// @Param stageId path string true "Stage identifier"
// @Param payload body createStageSessionRequest true "Session payload"
// @Success 201 {object} stageSessionResponse
// @Failure 400 {object} errmsg._StageInvalidRequest
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId}/sessions [post]
func CreateStageSessionHandler(c fiber.Ctx) error {
	stageID := strings.TrimSpace(c.Params("stageId"))
	if stageID == "" {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	var req createStageSessionRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	session, stage, err := core.SubmitStageSession(context.Background(), stageID, req.EnvText, req.Author, req.Notes, req.Source)
	if err != nil {
		return utils.StatusError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(stageSessionResponse{Stage: *stage, Session: *session})
}

// ListStageSessionsHandler returns all sessions for a stage.
// @Summary List stage sessions
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Produce json
// @Param stageId path string true "Stage identifier"
// @Success 200 {array} models.StageSession
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId}/sessions [get]
func ListStageSessionsHandler(c fiber.Ctx) error {
	stageID := strings.TrimSpace(c.Params("stageId"))
	if stageID == "" {
		return utils.StatusError(c, errmsg.StageNotFound)
	}

	if _, err := models.GetStageByID(context.Background(), stageID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return utils.StatusError(c, errmsg.StageNotFound)
		}
		return utils.StatusError(c, err)
	}

	sessions, err := models.ListStageSessions(context.Background(), stageID)
	if err != nil {
		return utils.StatusError(c, errmsg.InternalServerError(err))
	}

	return c.JSON(sessions)
}

type startStageTestRequest struct {
	SessionID string `json:"sessionId"`
}

// StartStageTestHandler launches a manual stage test run.
// @Summary Start stage test
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Accept json
// @Produce json
// @Param stageId path string true "Stage identifier"
// @Param payload body startStageTestRequest true "Test payload"
// @Success 201 {object} models.StageTestResult
// @Failure 400 {object} errmsg._StageInvalidRequest
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId}/tests [post]
func StartStageTestHandler(c fiber.Ctx) error {
	stageID := strings.TrimSpace(c.Params("stageId"))
	if stageID == "" {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	var req startStageTestRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		return utils.StatusError(c, errmsg.StageInvalidRequest)
	}

	result, err := core.StartStageTest(context.Background(), stageID, req.SessionID)
	if err != nil {
		return utils.StatusError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(result)
}

type promoteStageRequest struct {
	StageID string `json:"stageId"`
}

// CreateDeploymentHandler promotes a stage into a deployment record.
// @Summary Promote stage to deployment
// @Tags Hypervisor Deployments
// @Security HyperUserAuth
// @Accept json
// @Produce json
// @Param payload body promoteStageRequest true "Promotion payload"
// @Success 201 {object} models.Deployment
// @Failure 400 {object} errmsg._DeploymentInvalidRequest
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 409 {object} errmsg._StageMissingEnv
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/deployments [post]
func CreateDeploymentHandler(c fiber.Ctx) error {
	var req promoteStageRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return utils.StatusError(c, errmsg.DeploymentInvalidRequest)
	}

	req.StageID = strings.TrimSpace(req.StageID)
	if req.StageID == "" {
		return utils.StatusError(c, errmsg.DeploymentInvalidRequest)
	}

	dep, err := core.PromoteStage(context.Background(), req.StageID)
	if err != nil {
		return utils.StatusError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(dep)
}
