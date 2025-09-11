package utils

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func GetLocals(c fiber.Ctx, name string, result any) {
	json.Unmarshal(fmt.Appendf(nil, "%v", c.Locals(name)), &result)
}

func SetLocals(c fiber.Ctx, name string, data any) {
	bytes, _ := json.Marshal(data)
	json := string(bytes)
	c.Locals(name, json)
}
