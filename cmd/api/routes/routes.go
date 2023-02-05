package routes

import "github.com/go-redis/redis/v9"

func redisConn() {
	redis.NewClient()
}
