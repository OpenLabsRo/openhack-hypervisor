package models

import "time"

// GitCommit captures the push-commit fields stored from GitHub webhooks.
type GitCommit struct {
	DeliveryID string          `json:"delivery_id" bson:"delivery_id"`
	Ref        string          `json:"ref" bson:"ref"`
	SHA        string          `json:"sha" bson:"sha"`
	Message    string          `json:"message" bson:"message"`
	Author     GitCommitAuthor `json:"author" bson:"author"`
	Timestamp  time.Time       `json:"timestamp" bson:"timestamp"`
}

// GitCommitAuthor holds the commit author metadata recorded from hooks.
type GitCommitAuthor struct {
	Name  string `json:"name" bson:"name"`
	Email string `json:"email" bson:"email"`
}
