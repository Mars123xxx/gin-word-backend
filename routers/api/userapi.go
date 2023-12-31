package api

import (
	"awesomeProject/global"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"log"
	"strconv"
	"time"
)

func userIndex(c *gin.Context) {
	c.String(200, "userapi is ok")
}

func baseInfo(c *gin.Context) {
	db := global.MySQLDB
	currentUserId, exist := c.Get("userID")
	if exist {
		user := global.User{}
		db.First(&user, currentUserId)
		c.JSON(200, gin.H{
			"nickname": user.Username,
			"avatar":   user.Avatar,
			"schedule": user.Schedule,
		})
	} else {
		c.JSON(500, gin.H{
			"msg": "内部错误",
		})
	}
}

func studyInfo(c *gin.Context) {
	db := global.MySQLDB
	currentUserId, exist := c.Get("userID")
	if exist {
		startDate := time.Now().UTC().AddDate(0, 0, -6).Truncate(24 * time.Hour)
		var (
			learnedWordsCount   int64
			unlearnedWordsCount int64
			reviewWordsCount    int64
			allWordsCount       int64
		)
		type StudyData struct {
			Date      time.Time
			WordCount int
		}
		db.Where("user_id = ?", currentUserId).Find(&global.StudyLog{}).Count(&learnedWordsCount)
		db.Find(&global.Word{}).Count(&allWordsCount)
		unlearnedWordsCount = allWordsCount - learnedWordsCount
		db.Where("user_id = ? and next_study_time <= ?", currentUserId, time.Now().UTC()).Find(&global.StudyLog{}).Count(&reviewWordsCount)

		dateList := make([]time.Time, 7)
		for i := 0; i < 7; i++ {
			dateList[i] = startDate.AddDate(0, 0, i)
		}

		// 从StudyLog中筛选过去七天的数据，按照日期分组并计数
		var studyData []StudyData
		db.Model(&global.StudyLog{}).
			Select("DATE(start_study_time) as date, COUNT(word_id) as word_count").
			Where("DATE(start_study_time) >= ? AND user_id = ?", startDate, currentUserId).
			Group("DATE(start_study_time)").Scan(&studyData)

		// 将查询结果转换为map，便于查找和访问
		studyMap := make(map[time.Time]int)
		for _, record := range studyData {
			studyMap[record.Date] = record.WordCount
		}

		// 构建最终结果列表，确保包含所有日期
		var result []map[string]interface{}
		for _, date := range dateList {
			wordCount, ok := studyMap[date]
			if !ok {
				wordCount = 0
			}
			result = append(result, map[string]interface{}{"date": date, "word_count": wordCount})
		}
		c.JSON(200, gin.H{
			"learned_word_num": learnedWordsCount,
			"2learn_word_num":  unlearnedWordsCount,
			"2review_word_num": reviewWordsCount,
			"total_word_num":   allWordsCount,
			"days_study":       result,
		})
		return
	}
	c.JSON(200, gin.H{
		"status": "error",
	})
}

func changeSchedule(c *gin.Context) {
	db := global.MySQLDB
	currentUserId, exist := c.Get("userID")
	if exist {
		user := global.User{}
		db.First(&user, currentUserId)
		schedule, _ := strconv.Atoi(c.PostForm("schedule"))
		baseSchedule := []int{10, 15, 20}
		for i := 0; i < len(baseSchedule); i++ {
			if schedule == baseSchedule[i] {
				user.Schedule = schedule
				db.Save(&user)
				c.JSON(200, gin.H{
					"status": "success",
				})
				return
			}
		}
	}
	c.JSON(200, gin.H{
		"status": "error",
	})
}

func goStudy(c *gin.Context) {
	db := global.MySQLDB
	rdb := global.RedisDB
	currentUserId, exist := c.Get("userID")
	userId := currentUserId.(string)
	if exist {
		user := global.User{}
		db.First(&user, currentUserId)
		result, err := rdb.Exists("study:" + userId).Result()
		if err != nil {
			println("redis错误")
			return
		}
		if result != 0 {
			theFirstOneId, err := rdb.LIndex("study:"+userId, 0).Result()
			if err != nil {
				return
			}

			//这里返回一个map字典map[string]string
			theFirstOne, _ := rdb.HGetAll("study:" + userId + ":" + theFirstOneId).Result()
			// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
			deserializedMapping := make(map[string]interface{})

			// 遍历映射，并序列化每个值
			for k, v := range theFirstOne {
				var obj interface{}
				err := json.Unmarshal([]byte(v), &obj)
				if err != nil {
					log.Fatalf("JSON marshaling failed for key '%s': %s", k, err)
				}
				deserializedMapping[k] = obj
			}
			leftStudyCount, _ := rdb.LLen("study:" + userId).Result()
			deserializedMapping["left_study"] = strconv.Itoa(int(leftStudyCount))
			//返回一个对象
			c.JSONP(200, deserializedMapping)
			return
		} else {
			//没有缓存时加载缓存到redis中
			//TODO
		}
	}
	c.JSON(200, gin.H{
		"status": "error",
	})
}

func SetupUserRouter(userGroup *gin.RouterGroup) {
	userGroup.GET("/", userIndex)
	userGroup.POST("/baseInfo", baseInfo)
	userGroup.POST("/studyInfo", studyInfo)
	userGroup.POST("/schedule", changeSchedule)
	userGroup.POST("/goStudy", goStudy)
}
