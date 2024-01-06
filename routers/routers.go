package routers

import (
	"awesomeProject/global"
	apiBox "awesomeProject/routers/api"
	"awesomeProject/routers/middleware"
	"awesomeProject/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type SetupAPIRouterFunc func(*gin.RouterGroup)

func SetupRouter(c *gin.RouterGroup) {
	apiRouter := c.Group("/api")
	SetupAPIRouter(apiRouter)
	//获取验证码
	c.GET("/vCode", verifyCodeHandler)
	c.POST("/login", loginHandler)
}

// SetupAPIRouter 作为api接口的入口
func SetupAPIRouter(api *gin.RouterGroup) {
	api.Match([]string{"GET", "POST"}, "/", apiIndexHandler)

	api.Use(middleware.TokenAuthMiddleware())

	//定义功能模块
	apiRouterBox := map[string]SetupAPIRouterFunc{
		"/user":    apiBox.SetupUserRouter,
		"/word":    apiBox.SetupWordRouter,
		"/article": apiBox.SetupArticleRouter,
	}

	//通过循环批量注册路由
	for routerString, SetupFunc := range apiRouterBox {
		g := api.Group(routerString)
		SetupFunc(g)
	}
}

func apiIndexHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "api is ok",
	})
}

func verifyCodeHandler(c *gin.Context) {
	redisDB := global.RedisDB
	phoneNum, ok := c.GetQuery("phone_number")
	if !ok {
		return
	}
	exist, err := redisDB.Exists("verify_code:" + phoneNum).Result()
	if err != nil {
		println("查询Redis出错：", err)
		return
	}
	//如果该值存在
	if exist == 1 {
		c.JSON(200, gin.H{
			"msg": "Too Many Times Request",
		})
		return
	} else {
		if utils.IsValidChinaMobile(phoneNum) {
			var wg sync.WaitGroup
			rand.Seed(time.Now().UnixNano())
			wg.Add(1)
			code := rand.Intn(8888) + 1111
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				utils.SendSms(phoneNum, code)
			}(&wg)
			_, err := redisDB.Set("verify_code:"+phoneNum, code, time.Minute*5).Result()

			if err != nil {
				println("Redis插入错误", err)
				c.JSON(200, gin.H{"error": err})
				return
			}
			wg.Wait()
			c.JSON(200, gin.H{
				"status": "success",
			})
			return
		}
		c.JSON(200, gin.H{
			"status": "error",
			"msg":    "手机号格式错误",
		})
	}
}

func loginHandler(c *gin.Context) {
	db := global.MySQLDB
	redisDB := global.RedisDB
	phone := c.PostForm("phone_number")
	code := c.PostForm("code")
	if !utils.IsValidChinaMobile(phone) {
		c.JSON(200, gin.H{"error": "incorrect phone number"})
		return
	}
	redisCode, err := redisDB.Get("verify_code:" + phone).Result()
	if err != nil {
		c.JSON(200, gin.H{
			"error": "验证码已过期或失效",
		})
		return
	}
	if strings.TrimSpace(redisCode) != strings.TrimSpace(code) {
		c.JSON(200, gin.H{
			"error": "验证码错误",
		})
		return
	}
	user := global.User{}
	result := db.Where("Phone = ?", phone).Find(&user)
	// 检查错误
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newUser := global.User{Phone: phone, Username: "小芝麻_" + utils.GetRandomName(5)}
			db.Create(newUser)
			accessToken, _ := middleware.CreateAccessToken(newUser.ID)
			c.JSON(200, gin.H{
				"status":       "success",
				"access_token": accessToken,
			})
			return
		}
	} else {
		accessToken, _ := middleware.CreateAccessToken(user.ID)
		c.JSON(200, gin.H{
			"status":       "success",
			"access_token": accessToken,
		})
		return
	}
}
