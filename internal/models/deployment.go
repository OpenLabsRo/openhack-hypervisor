package models

import (
	"context"
	"hypervisor/internal/db"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type DeploymentStatus string

const (
	DeploymentStatusProvisioning    DeploymentStatus = "provisioning"
	DeploymentStatusReady           DeploymentStatus = "ready"
	DeploymentStatusStopped         DeploymentStatus = "stopped"
	DeploymentStatusBuildFailed     DeploymentStatus = "build_failed"
	DeploymentStatusProvisionFailed DeploymentStatus = "provision_failed"
)

// Deployment represents a running (or staged) instance of a release.
type Deployment struct {
	ID         string           `bson:"id" json:"id"`
	StageID    string           `bson:"stageId" json:"stageId"`
	Version    string           `bson:"version" json:"version"`
	EnvTag     string           `bson:"envTag" json:"envTag"`
	Port       *int             `bson:"port,omitempty" json:"port,omitempty"`
	Status     DeploymentStatus `bson:"status" json:"status"`
	LogPath    string           `bson:"logPath,omitempty" json:"logPath,omitempty"`
	CreatedAt  time.Time        `bson:"createdAt" json:"createdAt"`
	PromotedAt *time.Time       `bson:"promotedAt,omitempty" json:"promotedAt,omitempty"`
}

func CreateDeployment(ctx context.Context, dep Deployment) error {
	_, err := db.Deployments.InsertOne(ctx, dep)
	return err
}

func GetDeploymentByID(ctx context.Context, id string) (*Deployment, error) {
	var d Deployment
	err := db.Deployments.FindOne(ctx, bson.M{"id": id}).Decode(&d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func GetAllDeployments(ctx context.Context) ([]Deployment, error) {
	cursor, err := db.Deployments.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var deployments []Deployment
	if err := cursor.All(ctx, &deployments); err != nil {
		return nil, err
	}
	return deployments, nil
}

func UpdateDeployment(ctx context.Context, dep Deployment) error {
	_, err := db.Deployments.ReplaceOne(ctx, bson.M{"id": dep.ID}, dep)
	return err
}

func DeleteDeployment(ctx context.Context, id string) error {
	_, err := db.Deployments.DeleteOne(ctx, bson.M{"id": id})
	return err
}
