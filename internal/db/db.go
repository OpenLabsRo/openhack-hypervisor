package db

import (
	"context"
	"log"
	"strings"

	"hypervisor/internal/env"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Ctx    = context.Background()
	RDB    *redis.Client
	Client *mongo.Client

	HyperUsers *mongo.Collection
	GitCommits *mongo.Collection
	Events     *mongo.Collection
)

const databaseName = "hypervisor"

func InitDB(deployment string) error {
	var err error

	Client, err = mongo.Connect(
		Ctx,
		options.Client().ApplyURI(env.MONGO_URI),
	)
	if err != nil {
		return err
	}

	err = Client.Ping(Ctx, nil)
	if err != nil {
		log.Fatal("COULD NOT CONNECT TO MONGODB")
		return err
	}

	db := Client.Database(databaseName)
	HyperUsers = db.Collection("hyperusers")

	gitCommitCollection := "git_commits"
	eventsCollection := "events"

	if strings.EqualFold(strings.TrimSpace(deployment), "test") {
		// Use isolated collections so webhook tests do not mutate production data.
		gitCommitCollection = "test_git_commits"
		eventsCollection = "test_events"
	}

	GitCommits = db.Collection(gitCommitCollection)
	Events = db.Collection(eventsCollection)

	return nil
}

func InitCache() error {
	var err error

	RDB = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       15,
	})

	err = RDB.Ping(Ctx).Err()
	if err != nil {
		log.Fatal("COULD NOT CONNECT TO REDIS")
		return err
	}

	return nil
}

func CacheSet(key string, value string) error {
	return RDB.Set(Ctx, key, value, 0).Err()
}

func CacheSetBytes(key string, value []byte) error {
	return RDB.Set(Ctx, key, value, 0).Err()
}

func CacheGet(key string) (string, error) {
	return RDB.Get(Ctx, key).Result()
}

func CacheGetBytes(key string) ([]byte, error) {
	return RDB.Get(Ctx, key).Bytes()
}

func CacheDel(key string) error {
	_, err := RDB.Del(Ctx, key).Result()

	return err
}
