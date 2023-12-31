package middleware

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 一个简单的JWT密钥，实际生产中应该使用更安全的密钥，并且应该从配置中获取
var jwtSecretKey = []byte("sk-dahfa8798re324i289adgdfa&%^&5")

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中获取token
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证Token"})
			c.Abort()
			return
		}

		// 检查Bearer token格式
		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token格式"})
			c.Abort()
			return
		}

		// 解析Token
		token, err := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
			// 确保Token的签名方法是我们预期的
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecretKey, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Token验证通过，可以从claims提取用户信息，如用户ID等
			// 例如：userId := claims["id"].(string)
			c.Set("userID", claims["user_id"])
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
			c.Abort()
			return
		}

		c.Next() // 处理下一个中间件或者路由处理函数
	}
}

func CreateAccessToken(userId uint) (string, error) {
	// 创建一个新的Token对象，指定签名方法和claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": strconv.FormatUint(uint64(userId), 10),
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Token有效期72小时
	})

	// 使用密钥签名Token
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
