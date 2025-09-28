
package gitcommits

import (
	"context"
	"hypervisor/internal/db"
	"hypervisor/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

func Create(commit models.GitCommit) error {
	_, err := db.GitCommits.InsertOne(context.TODO(), commit)
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
