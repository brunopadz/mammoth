package load

import (
	"context"
	"errors"
	"github.com/brunopadz/mammoth/config"
	"github.com/go-redis/redis/v9"
)

func LoadProhibitedUsers() []string {
	var users []string
	ctx := context.Background()

	c := config.Config{}

	r := redis.NewClient(&redis.Options{
		Addr: c.RedisServer},
	)

	defer r.Close()

	u, err := r.SMembers(ctx, "users").Result()
	if err != nil {
		errors.New("Could not retrieve users from Redis")
	}

	for _, v := range u {
		users = append(users, v)
	}

	return users
}
