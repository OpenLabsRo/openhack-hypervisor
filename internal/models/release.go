package models

import (
	"context"
	"hypervisor/internal/db"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func ListReleases(ctx context.Context) ([]Release, error) {
	cursor, err := db.Releases.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var releases []Release
	if err := cursor.All(ctx, &releases); err != nil {
		return nil, err
	}

	return releases, nil
}
