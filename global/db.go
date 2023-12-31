package global

import (
	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

var (
	MySQLDB *gorm.DB
	RedisDB *redis.Client
)
