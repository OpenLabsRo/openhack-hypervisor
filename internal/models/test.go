package models

import (
	"context"
	"hypervisor/internal/db"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type TestStatus string

const (
	TestStatusRunning  TestStatus = "running"
	TestStatusPassed   TestStatus = "passed"
	TestStatusFailed   TestStatus = "failed"
	TestStatusCanceled TestStatus = "canceled"
	TestStatusError    TestStatus = "error"
)

type Test struct {
	ID         string     `bson:"id" json:"id"`
	StageID    string     `bson:"stageId" json:"stageId"`
	Status     TestStatus `bson:"status" json:"status"`
	WsToken    string     `bson:"wsToken" json:"wsToken"`
	LogPath    string     `bson:"logPath" json:"logPath"`
	StartedAt  time.Time  `bson:"startedAt" json:"startedAt"`
	FinishedAt *time.Time `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
	Error      string     `bson:"error,omitempty" json:"error,omitempty"`
}

func CreateTest(ctx context.Context, test Test) error {
	if test.StartedAt.IsZero() {
		test.StartedAt = time.Now().UTC()
	} else {
		test.StartedAt = test.StartedAt.UTC()
	}
	if test.FinishedAt != nil {
		t := test.FinishedAt.UTC()
		test.FinishedAt = &t
	}
	_, err := db.Tests.InsertOne(ctx, test)
	return err
}

func GetTestByID(ctx context.Context, id string) (*Test, error) {
	var test Test
	if err := db.Tests.FindOne(ctx, bson.M{"id": id}).Decode(&test); err != nil {
		return nil, err
	}
	return &test, nil
}

func UpdateTestStatus(ctx context.Context, id string, status TestStatus, finishedAt *time.Time, errMsg string) error {
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
	_, err := db.Tests.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": update})
	return err
}

func DeleteTestsByStageID(ctx context.Context, stageID string) ([]Test, error) {
	cursor, err := db.Tests.Find(ctx, bson.M{"stageId": stageID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tests []Test
	if err := cursor.All(ctx, &tests); err != nil {
		return nil, err
	}

	if _, err := db.Tests.DeleteMany(ctx, bson.M{"stageId": stageID}); err != nil {
		return nil, err
	}

	return tests, nil
}

func ListTestsByStageID(ctx context.Context, stageID string) ([]Test, error) {
	cursor, err := db.Tests.Find(ctx, bson.M{"stageId": stageID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tests []Test
	if err := cursor.All(ctx, &tests); err != nil {
		return nil, err
	}

	return tests, nil
}
