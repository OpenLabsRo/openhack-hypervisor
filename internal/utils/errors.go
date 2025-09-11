package utils

import (
	"github.com/gofiber/fiber/v3"
)

func Error(c fiber.Ctx, statusCode int, err error) error {
	return c.Status(statusCode).JSON(map[string]string{
		"message": err.Error(),
	})
}
