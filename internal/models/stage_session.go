package models

import (
	"context"
	"hypervisor/internal/db"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StageSession captures a single environment submission against a stage.
type StageSession struct {
	ID           string    `bson:"id" json:"id"`
	StageID      string    `bson:"stageId" json:"stageId"`
	EnvText      string    `bson:"envText" json:"envText"`
	Author       string    `bson:"author,omitempty" json:"author,omitempty"`
	Notes        string    `bson:"notes,omitempty" json:"notes,omitempty"`
	Source       string    `bson:"source" json:"source"` // template|manual|import
	TestResultID string    `bson:"testResultId,omitempty" json:"testResultId,omitempty"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
}

func CreateStageSession(ctx context.Context, session StageSession) error {
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	} else {
		session.CreatedAt = session.CreatedAt.UTC()
	}
	_, err := db.StageSessions.InsertOne(ctx, session)
	return err
}

func GetStageSessionByID(ctx context.Context, id string) (*StageSession, error) {
	var session StageSession
	if err := db.StageSessions.FindOne(ctx, bson.M{"id": id}).Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

func ListStageSessions(ctx context.Context, stageID string) ([]StageSession, error) {
	cursor, err := db.StageSessions.Find(ctx, bson.M{"stageId": stageID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []StageSession
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func UpdateStageSession(ctx context.Context, session StageSession) error {
	update := bson.M{
		"envText":      session.EnvText,
		"author":       session.Author,
		"notes":        session.Notes,
		"source":       session.Source,
		"testResultId": session.TestResultID,
	}
	_, err := db.StageSessions.UpdateOne(ctx, bson.M{"id": session.ID}, bson.M{"$set": update})
	return err
}
