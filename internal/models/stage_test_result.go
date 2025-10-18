package models

import (
	"context"
	"hypervisor/internal/db"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type StageTestStatus string

const (
	StageTestStatusRunning  StageTestStatus = "running"
	StageTestStatusPassed   StageTestStatus = "passed"
	StageTestStatusFailed   StageTestStatus = "failed"
	StageTestStatusCanceled StageTestStatus = "canceled"
	StageTestStatusError    StageTestStatus = "error"
)

type StageTestResult struct {
	ID         string          `bson:"id" json:"id"`
	StageID    string          `bson:"stageId" json:"stageId"`
	SessionID  string          `bson:"sessionId" json:"sessionId"`
	Status     StageTestStatus `bson:"status" json:"status"`
	WsToken    string          `bson:"wsToken" json:"wsToken"`
	LogPath    string          `bson:"logPath" json:"logPath"`
	StartedAt  time.Time       `bson:"startedAt" json:"startedAt"`
	FinishedAt *time.Time      `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
	Error      string          `bson:"error,omitempty" json:"error,omitempty"`
}

func CreateStageTestResult(ctx context.Context, result StageTestResult) error {
	if result.StartedAt.IsZero() {
		result.StartedAt = time.Now().UTC()
	} else {
		result.StartedAt = result.StartedAt.UTC()
	}
	if result.FinishedAt != nil {
		t := result.FinishedAt.UTC()
		result.FinishedAt = &t
	}
	_, err := db.StageTestResults.InsertOne(ctx, result)
	return err
}

func GetStageTestResultByID(ctx context.Context, id string) (*StageTestResult, error) {
	var result StageTestResult
	if err := db.StageTestResults.FindOne(ctx, bson.M{"id": id}).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func UpdateStageTestStatus(ctx context.Context, id string, status StageTestStatus, finishedAt *time.Time, errMsg string) error {
	update := bson.M{
		"status": status,
	}
	if finishedAt != nil {
		t := finishedAt.UTC()
		update["finishedAt"] = t
	}
	if errMsg != "" {
		update["error"] = errMsg
	}
	_, err := db.StageTestResults.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": update})
	return err
}
