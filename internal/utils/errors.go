package utils

import (
	"hypervisor/internal/errmsg"

	"github.com/gofiber/fiber/v3"
)

// MessageResponse represents a generic error response shape.
type MessageResponse struct {
	Message string `json:"message"`
}

func Error(c fiber.Ctx, statusCode int, err error) error {
	return c.Status(statusCode).JSON(MessageResponse{
		Message: err.Error(),
	})
}

func StatusError(c fiber.Ctx, err error) error {
	if se, ok := err.(errmsg.StatusError); ok {
		return c.Status(se.StatusCode).JSON(MessageResponse{
			Message: se.Message,
		})
	}
	return Error(c, fiber.StatusInternalServerError, err)
}
