package api

import (
	"awesomeProject/global"
	"database/sql"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"strconv"
	"sync"
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

			// 遍历映射，并反序列化每个值
			for k, v := range theFirstOne {
				var obj interface{}
				err := json.Unmarshal([]byte(v), &obj)
				if err != nil {
					return
				}
				deserializedMapping[k] = obj
			}
			leftStudyCount, _ := rdb.LLen("study:" + userId).Result()
			deserializedMapping["left_study"] = strconv.Itoa(int(leftStudyCount))
			//返回一个对象
			c.JSONP(200, deserializedMapping)
			return
		} else {
			var wordIDList []int
			//没有缓存时加载缓存到redis中
			rows, err := db.Raw("SELECT `id` FROM word WHERE NOT EXISTS(SELECT * FROM study_log where study_log.word_id = word.id and study_log.user_id = ? )", userId).Rows()
			if err != nil {
				return
			}
			defer func(rows *sql.Rows) {
				err := rows.Close()
				if err != nil {
					return
				}
			}(rows)
			for rows.Next() {
				var id int
				if err := rows.Scan(&id); err != nil {
					return
				}
				wordIDList = append(wordIDList, id)
			}
			if len(wordIDList) == 0 {
				c.JSON(200, gin.H{
					"msg": "暂无待学习的单词",
				})
				return
			}
			var idData []int
			if len(wordIDList) > user.Schedule {
				idData = wordIDList[0:user.Schedule]
			} else {
				idData = wordIDList
			}
			var wg sync.WaitGroup
			wg.Add(len(idData))
			//通过协程优化，这里为108ms
			for _, wid := range idData {
				go func(wid int) {
					defer wg.Done()
					item, _ := GetWordDetailByID(wid)
					rdb.LPush("study:"+userId, strconv.Itoa(wid))
					mapping := map[string]interface{}{
						"id":              item["id"],
						"word":            item["word"],
						"language":        item["language"],
						"sentences":       item["sentences"],
						"root_word":       item["root_word"],
						"root_meaning":    item["root_meaning"],
						"meanings":        item["meanings"],
						"collocations":    item["collocations"],
						"relative_words":  item["relative_words"],
						"right_option":    item["right_option"],
						"options":         item["options"],
						"similar_options": item["similar_options"],
						"times":           0,
						"total_study":     len(idData),
					}
					for k, v := range mapping {
						obj, err := json.Marshal(v)
						if err != nil {
							return
						}
						mapping[k] = obj
					}
					rdb.HMSet("study:"+userId+":"+strconv.Itoa(wid), mapping)
				}(wid)
			}
			//等待所有协程执行完毕
			wg.Wait()
			theFirstOneId, err := rdb.LIndex("study:"+userId, 0).Result()
			if err != nil {
				return
			}

			//这里返回一个map字典map[string]string
			theFirstOne, _ := rdb.HGetAll("study:" + userId + ":" + theFirstOneId).Result()
			// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
			deserializedMapping := make(map[string]interface{})

			// 遍历映射，并反序列化每个值
			for k, v := range theFirstOne {
				var obj interface{}
				err := json.Unmarshal([]byte(v), &obj)
				if err != nil {
					return
				}
				deserializedMapping[k] = obj
			}
			leftStudyCount, _ := rdb.LLen("study:" + userId).Result()
			deserializedMapping["left_study"] = strconv.Itoa(int(leftStudyCount))
			//返回一个对象
			c.JSONP(200, deserializedMapping)
			return
		}
	}
	c.JSON(200, gin.H{
		"status": "error",
	})
}

func goReview(c *gin.Context) {
	db := global.MySQLDB
	rdb := global.RedisDB
	currentUserId, exist := c.Get("userID")
	userId := currentUserId.(string)
	if exist {
		user := global.User{}
		db.First(&user, currentUserId)
		result, err := rdb.Exists("review:" + userId).Result()
		if err != nil {
			println("redis错误")
			return
		}
		if result != 0 {
			theFirstOneId, err := rdb.LIndex("review:"+userId, 0).Result()
			if err != nil {
				return
			}

			//这里返回一个map字典map[string]string
			theFirstOne, _ := rdb.HGetAll("review:" + userId + ":" + theFirstOneId).Result()
			// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
			deserializedMapping := make(map[string]interface{})

			// 遍历映射，并反序列化每个值
			for k, v := range theFirstOne {
				var obj interface{}
				err := json.Unmarshal([]byte(v), &obj)
				if err != nil {
					return
				}
				deserializedMapping[k] = obj
			}
			leftStudyCount, _ := rdb.LLen("review:" + userId).Result()
			deserializedMapping["left_review"] = strconv.Itoa(int(leftStudyCount))
			//返回一个对象
			c.JSONP(200, deserializedMapping)
			return
		} else {
			var wordIDList []int
			//没有缓存时加载缓存到redis中
			rows, err := db.Raw("SELECT word_id FROM study_log WHERE user_id = ? and next_study_time < ?", userId, time.Now()).Rows()
			if err != nil {
				return
			}
			defer func(rows *sql.Rows) {
				err := rows.Close()
				if err != nil {
					return
				}
			}(rows)
			for rows.Next() {
				var id int
				if err := rows.Scan(&id); err != nil {
					return
				}
				wordIDList = append(wordIDList, id)
			}
			if len(wordIDList) == 0 {
				c.JSON(200, gin.H{
					"msg": "暂无待复习的单词",
				})
				return
			}
			var idData []int
			if len(wordIDList) > user.Schedule {
				idData = wordIDList[0:user.Schedule]
			} else {
				idData = wordIDList
			}
			var wg sync.WaitGroup
			wg.Add(len(idData))
			//通过协程优化，这里为108ms
			for _, wid := range idData {
				go func(wid int) {
					defer wg.Done()
					item, _ := GetWordDetailByID(wid)
					rdb.LPush("review:"+userId, strconv.Itoa(wid))
					mapping := map[string]interface{}{
						"id":              item["id"],
						"word":            item["word"],
						"language":        item["language"],
						"sentences":       item["sentences"],
						"root_word":       item["root_word"],
						"root_meaning":    item["root_meaning"],
						"meanings":        item["meanings"],
						"collocations":    item["collocations"],
						"relative_words":  item["relative_words"],
						"right_option":    item["right_option"],
						"options":         item["options"],
						"similar_options": item["similar_options"],
						"times":           1,
						"total_review":    len(idData),
					}
					for k, v := range mapping {
						obj, err := json.Marshal(v)
						if err != nil {
							return
						}
						mapping[k] = obj
					}
					rdb.HMSet("review:"+userId+":"+strconv.Itoa(wid), mapping)
				}(wid)
			}
			//等待所有协程执行完毕
			wg.Wait()
			theFirstOneId, err := rdb.LIndex("review:"+userId, 0).Result()
			if err != nil {
				return
			}

			//这里返回一个map字典map[string]string
			theFirstOne, _ := rdb.HGetAll("review:" + userId + ":" + theFirstOneId).Result()
			// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
			deserializedMapping := make(map[string]interface{})

			// 遍历映射，并反序列化每个值
			for k, v := range theFirstOne {
				var obj interface{}
				err := json.Unmarshal([]byte(v), &obj)
				if err != nil {
					return
				}
				deserializedMapping[k] = obj
			}
			leftStudyCount, _ := rdb.LLen("review:" + userId).Result()
			deserializedMapping["left_review"] = strconv.Itoa(int(leftStudyCount))
			//返回一个对象
			c.JSONP(200, deserializedMapping)
			return
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
	userGroup.POST("/goReview", goReview)
}
