package models

import "time"

type Event struct {
	TimeStamp time.Time `json:"timestamp" bson:"timestamp"`

	Action string `bson:"action" json:"action"`

	ActorID   string `bson:"actorID" json:"actorID"`
	ActorRole string `bson:"actorRole" json:"actorRole"`

	TargetID   string `bson:"targetID" json:"targetID"`
	TargetType string `bson:"targetType" json:"targetType"`

	Props map[string]any `bson:"props" json:"props"`

	Key string `bson:"key,omitempty" json:"key,omitempty"`
}
