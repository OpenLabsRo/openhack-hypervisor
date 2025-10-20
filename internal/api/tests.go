package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"hypervisor/internal/core"
	"hypervisor/internal/models"
	"hypervisor/internal/utils"
	"hypervisor/internal/ws"

	"github.com/gofiber/fiber/v3"
)

var errClientClosed = errors.New("websocket closed by client")

// ListTestsHandler lists all tests for a stage.
// @Summary List tests
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Produce json
// @Param stageId path string true "Stage ID"
// @Success 200 {array} models.Test
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId}/tests [get]
func ListTestsHandler(c fiber.Ctx) error {
	stageID := c.Params("stageId")
	if stageID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing stage identifier")
	}

	tests, err := core.ListTests(context.Background(), stageID)
	if err != nil {
		return utils.StatusError(c, err)
	}

	return c.JSON(tests)
}

// StartTestHandler starts a new test run for a stage.
// @Summary Start test
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Produce json
// @Param stageId path string true "Stage ID"
// @Success 201 {object} models.Test
// @Failure 404 {object} errmsg._StageNotFound
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/stages/{stageId}/tests [post]
func StartTestHandler(c fiber.Ctx) error {
	stageID := c.Params("stageId")
	if stageID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing stage identifier")
	}

	test, err := core.StartTest(context.Background(), stageID)
	if err != nil {
		return utils.StatusError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(test)
}

// StreamTestLogs upgrades the connection and continuously streams a test log.
// @Summary Stream test logs
// @Tags Hypervisor Stages
// @Security HyperUserAuth
// @Param stageId path string true "Stage ID"
// @Param sequence path int true "Test sequence number"
// @Router /ws/stages/{stageId}/tests/{sequence} [get]
func StreamTestLogs(c fiber.Ctx) error {
	stageID := c.Params("stageId") // Note: This is validated by the test.StageID check below
	sequenceStr := c.Params("sequence")

	if stageID == "" || sequenceStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing stage or sequence identifier")
	}

	sequence := 0
	if _, err := fmt.Sscanf(sequenceStr, "%d", &sequence); err != nil || sequence <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sequence number")
	}

	testID := fmt.Sprintf("%s-test-%d", stageID, sequence)

	return ws.StreamWebSocket(c, func(ctx context.Context, writer *ws.WebsocketLogWriter) error {
		test, err := models.GetTestByID(ctx, testID)
		if err != nil {
			writer.WriteStatus("error", fmt.Sprintf("test not found: %v", err))
			return err
		}

		if test.StageID != stageID {
			writer.WriteStatus("error", "test does not belong to the requested stage")
			return errors.New("test does not belong to stage")
		}

		if test.LogPath == "" {
			writer.WriteStatus("error", "log path is not available for this test run")
			return errors.New("log path not available")
		}

		return core.StreamLogFile(ctx, test.LogPath, test.ID, writer, writer)
	})
}
