package utils

import (
	"regexp"
)

func IsValidChinaMobile(phoneNum string) bool {
	regex := `^1[3-9]\d{9}$`
	// 编译正则表达式
	re := regexp.MustCompile(regex)
	// 返回匹配结果
	return re.MatchString(phoneNum)
}
