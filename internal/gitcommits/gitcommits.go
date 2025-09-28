package gitcommits

import (
	"context"
	"hypervisor/internal/db"
	"hypervisor/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Create(commit models.GitCommit) error {
	filter := bson.M{
		"delivery_id": commit.DeliveryID,
		"sha":         commit.SHA,
	}

	update := bson.M{
		"$setOnInsert": commit,
	}

	_, err := db.GitCommits.UpdateOne(context.TODO(), filter, update, options.Update().SetUpsert(true))
	return err
}

func GetAll() ([]models.GitCommit, error) {
	var commits []models.GitCommit
	cursor, err := db.GitCommits.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &commits); err != nil {
		return nil, err
	}
	return commits, nil
}
