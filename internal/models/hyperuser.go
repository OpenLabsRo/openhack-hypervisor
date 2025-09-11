package models

import (
	"errors"
	"hypervisor/internal/db"
	"hypervisor/internal/env"
	"hypervisor/internal/utils"
	"net/http"
	"strings"
	"time"

	sj "github.com/brianvoe/sjwt"
	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/bson"
)

type HyperUser struct {
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
}

func (hu *HyperUser) GenToken() string {
	claims, _ := sj.ToClaims(hu)
	claims.SetExpiresAt(time.Now().Add(365 * 24 * time.Hour))

	token := claims.Generate(env.JWT_SECRET)
	return token
}

func (hu *HyperUser) ParseToken(token string) error {
	hasVerified := sj.Verify(token, env.JWT_SECRET)

	if !hasVerified {
		return nil
	}

	claims, _ := sj.Parse(token)
	err := claims.Validate()
	claims.ToStruct(&hu)

	return err
}

func AccountMiddleware(c fiber.Ctx) error {
	var token string

	authHeader := c.Get("Authorization")

	if string(authHeader) != "" &&
		strings.HasPrefix(string(authHeader), "Bearer") {

		tokens := strings.Fields(string(authHeader))
		if len(tokens) == 2 {
			token = tokens[1]
		}

		if token == "" {
			return utils.Error(
				c,
				http.StatusUnauthorized,
				errors.New("unauthorized"),
			)
		}

		var hyperuser HyperUser
		err := hyperuser.ParseToken(token)
		if err != nil {
			return errors.New("unauthorized")
		}

		if hyperuser.Username == "" {
			return utils.Error(
				c,
				http.StatusUnauthorized,
				errors.New("unauthorized"),
			)
		}

		utils.SetLocals(c, "hyperuser", hyperuser)
	}

	if token == "" {
		return utils.Error(
			c,
			http.StatusUnauthorized,
			errors.New("unauthorized"),
		)
	}

	return c.Next()
}

func (hu *HyperUser) Get(username string) (err error) {
	err = db.HyperUsers.FindOne(db.Ctx, bson.M{
		"username": username,
	}).Decode(&hu)
	if err != nil {
		return err
	}

	if hu.Password == "" {
		return errors.New("hyperuser does not exist")
	}

	return
}
