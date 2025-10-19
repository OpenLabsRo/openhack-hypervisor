package hyperusers

import (
	"encoding/json"
	"strings"

	"hypervisor/internal/errmsg"
	"hypervisor/internal/events"
	"hypervisor/internal/models"
	"hypervisor/internal/utils"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

type HyperUserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type HyperUserLoginResponse struct {
	Token     string           `json:"token"`
	Hyperuser models.HyperUser `json:"hyperuser"`
}

// loginHandler authenticates hyperusers and returns a scoped JWT token.
// @Summary Hyperuser login
// @Description Validates hyperuser credentials and issues a 24-hour bearer token.
// @Tags Hyperusers Auth
// @Accept json
// @Produce json
// @Param payload body HyperUserLoginRequest true "Login credentials"
// @Success 200 {object} HyperUserLoginResponse
// @Failure 400 {object} errmsg._HyperUserInvalidPayload
// @Failure 401 {object} errmsg._HyperUserWrongPassword
// @Failure 404 {object} errmsg._HyperUserNotExists
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/hyperusers/login [post]
func loginHandler(c fiber.Ctx) error {
	var body HyperUserLoginRequest
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

	return c.JSON(HyperUserLoginResponse{
		Token:     token,
		Hyperuser: hu,
	})
}
