package api

import (
	"encoding/json"

	"hypervisor/internal/core"
	"hypervisor/internal/errmsg"
	"hypervisor/internal/utils"

	"github.com/gofiber/fiber/v3"
)

type envTemplateResponse struct {
	EnvText string `json:"envText"`
}

type updateEnvTemplateRequest struct {
	EnvText *string `json:"envText"`
}

// GetEnvTemplateHandler returns the current OpenHack backend template .env contents.
// @Summary Get template env
// @Tags Hypervisor Env
// @Security HyperUserAuth
// @Produce json
// @Success 200 {object} envTemplateResponse
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/env/template [get]
func GetEnvTemplateHandler(c fiber.Ctx) error {
	envText, err := core.ReadEnvTemplate()
	if err != nil {
		return utils.StatusError(c, errmsg.InternalServerError(err))
	}

	return c.JSON(envTemplateResponse{EnvText: envText})
}

// UpdateEnvTemplateHandler replaces the template .env file with the provided contents.
// @Summary Update template env
// @Tags Hypervisor Env
// @Security HyperUserAuth
// @Accept json
// @Produce json
// @Param payload body updateEnvTemplateRequest true "Template env contents"
// @Success 200 {object} StatusResponse
// @Failure 400 {object} errmsg._EnvTemplateInvalidRequest
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/env/template [put]
func UpdateEnvTemplateHandler(c fiber.Ctx) error {
	var payload updateEnvTemplateRequest
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return utils.StatusError(c, errmsg.EnvTemplateInvalidRequest)
	}

	if payload.EnvText == nil {
		return utils.StatusError(c, errmsg.EnvTemplateInvalidRequest)
	}

	if err := core.WriteEnvTemplate(*payload.EnvText); err != nil {
		return utils.StatusError(c, errmsg.InternalServerError(err))
	}

	return c.JSON(StatusResponse{Status: "template updated"})
}
