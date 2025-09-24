package utils

import (
	"hypervisor/internal/errmsg"

	"github.com/gofiber/fiber/v3"
)

func Error(c fiber.Ctx, statusCode int, err error) error {
	return c.Status(statusCode).JSON(map[string]string{
		"message": err.Error(),
	})
}

func StatusError(c fiber.Ctx, se errmsg.StatusError) error {
	return c.Status(se.StatusCode).JSON(map[string]string{
		"message": se.Message,
	})
}
