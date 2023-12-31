package utils

import (
	"math/rand"
	"strings"
	"time"
)

func GetRandomName(length int) string {
	rand.Seed(time.Now().UnixNano())
	// 定义可用的字符集
	charSet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// 生成随机名称
	var sb strings.Builder
	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(charSet))
		randomChar := charSet[randomIndex]
		sb.WriteByte(randomChar)
	}

	return sb.String()
}
