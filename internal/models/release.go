
package models

import "time"

// Release represents a version of the application that can be staged and deployed.
type Release struct {
	Tag       string    `bson:"tag" json:"tag"`
	Status    string    `bson:"status" json:"status"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
