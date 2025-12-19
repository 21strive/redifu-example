package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
)

func ConnectRedis(host string, username string, password string, isClustered bool) redis.UniversalClient {
	if host == "" {
		log.Fatal("REDIS_HOST environment variable not set")
	}

	if isClustered {
		clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{host},
			Username: username,
			Password: password,
		})

		_, err := clusterClient.Ping(context.Background()).Result()
		if err != nil {
			log.Fatal(err)
		}

		return clusterClient
	}

	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Username: username,
		Password: password,
		DB:       0,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	return client
}
