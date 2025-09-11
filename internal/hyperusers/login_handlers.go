package hyperusers

import (
	"encoding/json"
	"hypervisor/internal/models"
	"hypervisor/internal/utils"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func loginHandler(c fiber.Ctx) error {
	var body models.HyperUser
	json.Unmarshal(c.Body(), &body)

	hu := models.HyperUser{}
	err := hu.Get(body.Username)
	if err != nil {
		return utils.Error(
			c,
			500,
			err,
		)
	}

	if bcrypt.CompareHashAndPassword(
		[]byte(hu.Password),
		[]byte(body.Password),
	) != nil {
		return utils.Error(
			c,
			500,
			err,
		)
	}

	token := hu.GenToken()

	return c.JSON(bson.M{
		"token":     token,
		"hyperuser": hu,
	})
}
