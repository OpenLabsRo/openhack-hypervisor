package models

import (
	"context"
	"hypervisor/internal/db"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StageStatus string

const (
	StageStatusPre      StageStatus = "pre"
	StageStatusReady    StageStatus = "ready"
	StageStatusPromoted StageStatus = "promoted"
)

// Stage represents the configuration workspace for a release/environment pair.
type Stage struct {
	ID           string      `bson:"id" json:"id"`
	ReleaseID    string      `bson:"releaseId" json:"releaseId"`
	EnvTag       string      `bson:"envTag" json:"envTag"`
	Status       StageStatus `bson:"status" json:"status"`
	TestSequence int         `bson:"testSequence,omitempty" json:"testSequence,omitempty"`
	CreatedAt    time.Time   `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time   `bson:"updatedAt" json:"updatedAt"`
}

func CreateStage(ctx context.Context, stage Stage) error {
	now := time.Now().UTC()
	if stage.CreatedAt.IsZero() {
		stage.CreatedAt = now
	} else {
		stage.CreatedAt = stage.CreatedAt.UTC()
	}
	if stage.UpdatedAt.IsZero() {
		stage.UpdatedAt = now
	} else {
		stage.UpdatedAt = stage.UpdatedAt.UTC()
	}
	stage.TestSequence = 0
	_, err := db.Stages.InsertOne(ctx, stage)
	return err
}

func GetStageByID(ctx context.Context, id string) (*Stage, error) {
	var stage Stage
	err := db.Stages.FindOne(ctx, bson.M{"id": id}).Decode(&stage)
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

func UpdateStage(ctx context.Context, stage Stage) error {
	_, err := db.Stages.UpdateOne(ctx, bson.M{"id": stage.ID}, bson.M{
		"$set": bson.M{
			"status":       stage.Status,
			"testSequence": stage.TestSequence,
			"updatedAt":    stage.UpdatedAt.UTC(),
		},
	})
	return err
}

func DeleteStage(ctx context.Context, stageID string) error {
	_, err := db.Stages.DeleteOne(ctx, bson.M{"id": stageID})
	return err
}

// NextTestSequence increments the stage test sequence counter and returns the new value.
func NextTestSequence(ctx context.Context, stageID string) (int, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	update := bson.M{
		"$inc": bson.M{"testSequence": 1},
	}

	var stage Stage
	if err := db.Stages.FindOneAndUpdate(ctx, bson.M{"id": stageID}, update, opts).Decode(&stage); err != nil {
		return 0, err
	}

	return stage.TestSequence, nil
}

func ListStages(ctx context.Context) ([]Stage, error) {
	cursor, err := db.Stages.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stages []Stage
	if err := cursor.All(ctx, &stages); err != nil {
		return nil, err
	}
	return stages, nil
}
