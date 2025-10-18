package models

import (
	"context"
	"hypervisor/internal/db"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Release represents an immutable build artifact of a tagged commit.
type Release struct {
	ID        string    `bson:"id" json:"id"`
	Sha       string    `bson:"sha" json:"sha"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

func CreateRelease(ctx context.Context, release Release) error {
	_, err := db.Releases.InsertOne(ctx, release)
	return err
}

func GetReleaseByID(ctx context.Context, id string) (*Release, error) {
	var r Release
	err := db.Releases.FindOne(ctx, bson.M{"id": id}).Decode(&r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
