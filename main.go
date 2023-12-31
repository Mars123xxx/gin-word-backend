package main

import (
	"awesomeProject/global"
	"awesomeProject/routers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"net/http"
	"time"
)

func main() {
	//连接数据库
	dsn := "root:abc123@tcp(127.0.0.1:3306)/db_word?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",   // 表名前缀，可根据需要设置
			SingularTable: true, // 使用单数表名
		},
	})
	if err != nil {
		panic("failed to connect database")
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     "49.232.235.202:6379",
		Password: "taotaohan0402",
		DB:       1,
	})
	global.RedisDB = rdb
	global.MySQLDB = db
	rootRouter := gin.Default()
	// 自定义CORS配置
	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // 这里应该修改为你的前端服务器地址
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://localhost:5173" // 或者使用更灵活的匹配规则
		},
		MaxAge: 12 * time.Hour,
	}
	rootRouter.Use(cors.New(config))
	routers.SetupRouter(rootRouter)
	rootRouter.GET("/", indexHandler)
	err = rootRouter.Run(":8089")
	if err != nil {
		return
	}
}

func indexHandler(r *gin.Context) {
	r.JSON(http.StatusOK, gin.H{
		"message":     "ok",
		"status_code": 200,
	})
}
