
package db

import (
	"context"
	"hypervisor/internal/db"
	"hypervisor/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

func Create(release models.Release) error {
	_, err := db.Releases.InsertOne(context.TODO(), release)
	return err
}

func GetAll() ([]models.Release, error) {
	var releases []models.Release
	cursor, err := db.Releases.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func UpdateStatus(tag, status string) error {
	filter := bson.M{"tag": tag}
	update := bson.M{"$set": bson.M{"status": status}}
	_, err := db.Releases.UpdateOne(context.TODO(), filter, update)
	return err
}
