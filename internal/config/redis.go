package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis() {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	password := os.Getenv("REDIS_PASSWORD")

	db := 0
	if raw := os.Getenv("REDIS_DB"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			db = parsed
		}
	}

	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := RDB.Ping(ctx).Err(); err != nil {
		log.Fatal("cannot connect redis: ", err)
	}
}
