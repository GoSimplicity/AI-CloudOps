package di

import (
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	// 初始化 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: viper.GetString("redis.addr"),
	})

	return client
}
