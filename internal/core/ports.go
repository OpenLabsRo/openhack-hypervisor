package core

import (
	"context"
	"fmt"
	"hypervisor/internal/db"
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
)

// AllocatePort finds an available port in the configured range.
func AllocatePort(ctx context.Context) (int, error) {
	start, err := strconv.Atoi(os.Getenv("PORT_RANGE_START"))
	if err != nil {
		start = 20000
	}
	end, err := strconv.Atoi(os.Getenv("PORT_RANGE_END"))
	if err != nil {
		end = 29999
	}

	// Get all used ports
	cursor, err := db.Deployments.Find(ctx, bson.M{"port": bson.M{"$ne": nil}})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	usedPorts := make(map[int]bool)
	for cursor.Next(ctx) {
		var dep struct {
			Port *int `bson:"port"`
		}
		if err := cursor.Decode(&dep); err != nil {
			return 0, err
		}
		if dep.Port != nil {
			usedPorts[*dep.Port] = true
		}
	}

	// Find first available
	for port := start; port <= end; port++ {
		if !usedPorts[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", start, end)
}
