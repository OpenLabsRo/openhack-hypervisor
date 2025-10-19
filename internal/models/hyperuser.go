package models

import (
	"errors"
	"hypervisor/internal/db"
	"hypervisor/internal/env"
	"hypervisor/internal/errmsg"
	"hypervisor/internal/utils"
	"strings"
	"time"

	sj "github.com/brianvoe/sjwt"
	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type HyperUser struct {
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
}

func (hu *HyperUser) GenToken() string {
	claims, _ := sj.ToClaims(hu)
	claims.SetExpiresAt(time.Now().Add(24 * time.Hour))

	token := claims.Generate(env.JWT_SECRET)
	return token
}

func (hu *HyperUser) ParseToken(token string) error {
	hasVerified := sj.Verify(token, env.JWT_SECRET)
	if !hasVerified {
		return errors.New("invalid token")
	}

	claims, err := sj.Parse(token)
	if err != nil {
		return err
	}

	if err := claims.Validate(); err != nil {
		return err
	}

	claims.ToStruct(hu)

	if strings.TrimSpace(hu.Username) == "" {
		return errors.New("invalid token")
	}

	return nil
}

func HyperUserMiddleware(c fiber.Ctx) error {
	authHeader := strings.TrimSpace(c.Get("Authorization"))
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}

	tokens := strings.Fields(authHeader)
	if len(tokens) != 2 {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}

	token := strings.TrimSpace(tokens[1])
	if token == "" {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}

	var hyperuser HyperUser
	if err := hyperuser.ParseToken(token); err != nil {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}

	utils.SetLocals(c, "hyperuser", hyperuser)

	return c.Next()
}

func (hu *HyperUser) Get(username string) (serr errmsg.StatusError) {
	serr = errmsg.EmptyStatusError

	err := db.HyperUsers.FindOne(db.Ctx, bson.M{
		"username": username,
	}).Decode(hu)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errmsg.HyperUserNotExists
		}

		return errmsg.InternalServerError(err)
	}

	if strings.TrimSpace(hu.Password) == "" {
		return errmsg.HyperUserNotExists
	}

	return serr
}
