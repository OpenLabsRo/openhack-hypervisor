package hyperusers

import (
	"encoding/json"
	"hypervisor/internal/errmsg"
	"hypervisor/internal/events"
	"hypervisor/internal/models"
	"hypervisor/internal/utils"
	"strings"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func loginHandler(c fiber.Ctx) error {
	var body models.HyperUser
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return utils.StatusError(c, errmsg.HyperUserInvalidPayload)
	}

	body.Username = strings.TrimSpace(body.Username)
	body.Password = strings.TrimSpace(body.Password)
	if body.Username == "" || body.Password == "" {
		return utils.StatusError(c, errmsg.HyperUserInvalidPayload)
	}

	hu := models.HyperUser{}
	serr := hu.Get(body.Username)
	if serr != errmsg.EmptyStatusError {
		return utils.StatusError(c, serr)
	}

	if bcrypt.CompareHashAndPassword(
		[]byte(hu.Password),
		[]byte(body.Password),
	) != nil {
		return utils.StatusError(c, errmsg.HyperUserWrongPassword)
	}

	token := hu.GenToken()

	if events.Em != nil {
		events.Em.HyperUserLogin(hu.Username)
	}

	hu.Password = ""

	return c.JSON(bson.M{
		"token":     token,
		"hyperuser": hu,
	})
}
