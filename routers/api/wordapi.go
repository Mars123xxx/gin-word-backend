package api

import (
	"awesomeProject/global"
	"awesomeProject/utils"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"math/rand"
	"strconv"
	"time"
)

func wordIndex(c *gin.Context) {
	c.String(200, "wordapi is ok")
}

func GetWordBaseDetailByID(id int) map[string]interface{} {
	db := global.MySQLDB
	word := global.Word{}
	db.First(&word, id)
	var (
		sentences, meanings, collocations, relatives []map[string]interface{}
	)
	db.Model(&global.Sentence{}).
		Select("content, word_id as from_word_id").
		Where("word_id = ?", id).Scan(&sentences)
	db.Model(&global.Meaning{}).
		Select("part_of_speech, definition, word_id as from_word_id").
		Where("word_id = ?", id).Scan(&meanings)
	db.Model(&global.Collocation{}).
		Select("content, word_id as from_word_id").
		Where("word_id = ?", id).Scan(&collocations)
	db.Model(&global.Relative{}).
		Select("content, word_id as from_word_id").
		Where("word_id = ?", id).Scan(&relatives)
	baseDetail := map[string]interface{}{
		"id":           id,
		"word":         word.Word,
		"language":     word.Language,
		"root_word":    word.RootWord,
		"root_meaning": word.RootMeaning,
		"sentences":    sentences,
		"meanings":     meanings,
		"collocations": collocations,
		"relatives":    relatives,
	}
	return baseDetail
}

func GetWordDetailByID(id int) (map[string]interface{}, error) {
	db := global.MySQLDB
	baseDetail := GetWordBaseDetailByID(id)
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())
	result, ok := baseDetail["meanings"]
	var (
		rightOption    map[string]interface{}
		options        []map[string]interface{}
		similarOptions []string
	)
	if ok {
		// 生成一个随机索引
		randomIndex := rand.Intn(len(result.([]map[string]interface{})))
		rightOption = result.([]map[string]interface{})[randomIndex]
	} else {
		return nil, errors.New("key is not found")
	}

	//对baseDetail进行解包
	wordDetail := make(map[string]interface{})
	for k, v := range baseDetail {
		wordDetail[k] = v
	}
	wordDetail["right_option"] = rightOption

	db.Model(&global.Meaning{}).
		Select("part_of_speech, definition").
		Where("word_id != ?", id).Order("RAND()").Limit(3).Scan(&options)
	db.Model(&global.Looklike{}).
		Select("word").
		Where("word_id = ?", id).Scan(&similarOptions)

	wordDetail["options"] = options
	wordDetail["similar_options"] = similarOptions
	return wordDetail, nil
}

func wordDetail(wordDetailType string, c *gin.Context) map[string]interface{} {
	id, _ := strconv.Atoi(c.Param("id"))
	rdb := global.RedisDB
	currentUserId, exist := c.Get("userID")
	if !exist {
		return nil
	}
	userId := currentUserId.(string)
	result, err := rdb.Exists(wordDetailType + ":" + userId + ":" + strconv.Itoa(id)).Result()
	if err != nil {
		println("redis错误")
		return nil
	}
	if result != 0 {
		//这里返回一个map字典map[string]string
		data, _ := rdb.HGetAll(wordDetailType + ":" + userId + ":" + strconv.Itoa(id)).Result()
		// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
		deserializedMapping := make(map[string]interface{})

		// 遍历映射，并序列化每个值
		for k, v := range data {
			var obj interface{}
			err := json.Unmarshal([]byte(v), &obj)
			if err != nil {
				return nil
			}
			deserializedMapping[k] = obj
		}
		return deserializedMapping
	}
	wordDetail, err := GetWordDetailByID(id)
	if err != nil {
		return nil
	}
	serializedMapping := make(map[string]interface{})
	for k, v := range wordDetail {
		// 遍历映射，并序列化每个值
		obj, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		serializedMapping[k] = obj
	}
	rdb.HMSet(wordDetailType+":"+userId+":"+strconv.Itoa(id), serializedMapping)
	return wordDetail
}

func wordStudyDetail(c *gin.Context) {
	studyDetail := wordDetail("study", c)
	if studyDetail == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, studyDetail)
}

func wordReviewDetail(c *gin.Context) {
	reviewDetail := wordDetail("review", c)
	if reviewDetail == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, reviewDetail)
}

func wordYes(wordYesType string, c *gin.Context) map[string]interface{} {
	id, _ := strconv.Atoi(c.Param("id"))
	db := global.MySQLDB
	rdb := global.RedisDB
	currentUserId, exist := c.Get("userID")
	if !exist {
		return nil
	}
	userId := currentUserId.(string)
	//这个flag用于判断是否要记录该单词的学习次数
	var flag bool
	flag_, ok := c.GetPostForm("flag")
	if ok {
		flag, _ = strconv.ParseBool(flag_)
	} else {
		flag = false
	}
	stackUserKey := wordYesType + ":" + userId
	wordUserKey := stackUserKey + ":" + strconv.Itoa(id)

	result1, err := rdb.Exists(stackUserKey).Result()
	result2, err := rdb.Exists(wordUserKey).Result()
	if err != nil {
		println("redis错误")
		return nil
	}
	if result1&result2 == 1 {
		studyTimes, _ := rdb.HGet(wordUserKey, "times").Result()
		if studyTimes, _ := strconv.Atoi(studyTimes); studyTimes >= 2 {
			topWordId, _ := rdb.LPop(stackUserKey).Result()
			// 这里需要在数据库中更新用户的学习记录，并且通过算法生成该用户对该单词学习的下一时间
			today := time.Now()
			uId, err := strconv.Atoi(userId)
			wId, err := strconv.Atoi(topWordId)
			if err != nil {
				return nil
			}
			switch wordYesType {
			case "study":
				nextReviewDate := utils.GetNextReviewDate(today, 0)
				studyLog := global.StudyLog{UserID: uint(uId), WordID: uint(wId), NextStudyTime: nextReviewDate}
				db.Create(&studyLog)
			case "review":
				uId, _ = strconv.Atoi(userId)
				log := global.StudyLog{UserID: uint(uId), WordID: uint(id)}
				if err := db.First(&log).Error; errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				nextReviewDate := utils.GetNextReviewDate(today, log.StudyTimes)
				studyLog := global.StudyLog{UserID: uint(uId), WordID: uint(wId), NextStudyTime: nextReviewDate, StudyTimes: log.StudyTimes + 1}
				db.Create(&studyLog)
			default:
				return nil
			}

		} else {
			//在原来基础上+1
			rdb.HIncrBy(wordUserKey, "times", 1)
			//从左边pop出来，然后push到右边的队尾
			topWordId, _ := rdb.LPop(stackUserKey).Result()
			//获取List的长度
			stackLen, _ := rdb.LLen(stackUserKey).Result()
			if stackLen > 1 {
				rand.Seed(time.Now().UnixNano())
				randomIndex := rand.Intn(int(stackLen))
				// 获取随机索引位置的元素
				pivot, _ := rdb.LIndex(stackUserKey, int64(randomIndex)).Result()
				// 向List的随机位置插入数据
				rdb.LInsertBefore(stackUserKey, pivot, topWordId)
			} else {
				rdb.RPush(stackUserKey, topWordId)
			}
		}
		if flag {
			topWordId, _ := rdb.LIndex(stackUserKey, 0).Result()
			topWordItem, _ := rdb.HGetAll(stackUserKey + ":" + topWordId).Result()
			// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
			deserializedMapping := make(map[string]interface{})

			// 遍历映射，并序列化每个值
			for k, v := range topWordItem {
				var obj interface{}
				err := json.Unmarshal([]byte(v), &obj)
				if err != nil {
					return nil
				}
				deserializedMapping[k] = obj
			}
			//获取List的长度
			stackLen, _ := rdb.LLen(stackUserKey).Result()
			deserializedMapping["left_study"] = stackLen
			return deserializedMapping
		}
		return map[string]interface{}{
			"status": "success",
		}
	}
	println(5)
	return nil
}

func studyWordYes(c *gin.Context) {
	studyWordYesResult := wordYes("study", c)
	if studyWordYesResult == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, studyWordYesResult)
}

func reviewWordYes(c *gin.Context) {
	reviewWordYesResult := wordYes("review", c)
	if reviewWordYesResult == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, reviewWordYesResult)
}

func btnNext(wordType string, c *gin.Context) map[string]interface{} {
	id, _ := strconv.Atoi(c.Param("id"))
	rdb := global.RedisDB
	db := global.MySQLDB
	currentUserId, exist := c.Get("userID")
	if !exist {
		return nil
	}
	userId := currentUserId.(string)
	stackUserKey := wordType + ":" + userId
	wordUserKey := stackUserKey + ":" + strconv.Itoa(id)
	result1, err := rdb.Exists(stackUserKey).Result()
	result2, err := rdb.Exists(wordUserKey).Result()
	if err != nil {
		println("redis错误")
		return nil
	}
	if result1&result2 == 1 {
		studyTimes, _ := rdb.HGet(wordUserKey, "times").Result()
		if studyTimes, _ := strconv.Atoi(studyTimes); studyTimes >= 2 {
			topWordId, _ := rdb.LPop(stackUserKey).Result()
			// 这里需要在数据库中更新用户的学习记录，并且通过算法生成该用户对该单词学习的下一时间
			today := time.Now()
			uId, err := strconv.Atoi(userId)
			wId, err := strconv.Atoi(topWordId)
			if err != nil {
				return nil
			}
			nextReviewDate := utils.GetNextReviewDate(today, 0)
			studyLog := global.StudyLog{UserID: uint(uId), WordID: uint(wId), NextStudyTime: nextReviewDate}
			db.Create(&studyLog)
		}
		//获取List的长度
		stackLen, _ := rdb.LLen(stackUserKey).Result()
		topWordId, _ := rdb.LIndex(stackUserKey, 0).Result()
		topWordItem, _ := rdb.HGetAll(stackUserKey + ":" + topWordId).Result()
		// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
		deserializedMapping := make(map[string]interface{})

		// 遍历映射，并序列化每个值
		for k, v := range topWordItem {
			var obj interface{}
			err := json.Unmarshal([]byte(v), &obj)
			if err != nil {
				return nil
			}
			deserializedMapping[k] = obj
		}
		deserializedMapping["left_study"] = stackLen
		return deserializedMapping
	}
	return nil
}

func studyBtnNext(c *gin.Context) {
	studyBtnNextResult := btnNext("study", c)
	if studyBtnNextResult == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, studyBtnNextResult)
}

func reviewBtnNext(c *gin.Context) {
	reviewBtnNextResult := btnNext("review", c)
	if reviewBtnNextResult == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, reviewBtnNextResult)
}

func btnError(wordType string, c *gin.Context) map[string]interface{} {
	id, _ := strconv.Atoi(c.Param("id"))
	rdb := global.RedisDB
	currentUserId, exist := c.Get("userID")
	if !exist {
		return nil
	}
	userId := currentUserId.(string)
	stackUserKey := wordType + ":" + userId
	wordUserKey := stackUserKey + ":" + strconv.Itoa(id)
	result, _ := rdb.Exists(wordUserKey).Result()
	if result != 0 {
		rdb.HSet(wordUserKey, "times", 0)
		topWordId, _ := rdb.LPop(stackUserKey).Result()
		stackLen, _ := rdb.LLen(stackUserKey).Result()
		if stackLen > 1 {
			rand.Seed(time.Now().UnixNano())
			randomIndex := rand.Intn(int(stackLen))
			// 获取随机索引位置的元素
			pivot, _ := rdb.LIndex(stackUserKey, int64(randomIndex)).Result()
			// 向List的随机位置插入数据
			rdb.LInsertBefore(stackUserKey, pivot, topWordId)
		} else {
			rdb.RPush(stackUserKey, topWordId)
		}
		//获取List的长度
		stackLen, _ = rdb.LLen(stackUserKey).Result()
		topWordId, _ = rdb.LIndex(stackUserKey, 0).Result()
		topWordItem, _ := rdb.HGetAll(stackUserKey + ":" + topWordId).Result()
		// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
		deserializedMapping := make(map[string]interface{})

		// 遍历映射，并序列化每个值
		for k, v := range topWordItem {
			var obj interface{}
			err := json.Unmarshal([]byte(v), &obj)
			if err != nil {
				return nil
			}
			deserializedMapping[k] = obj
		}
		deserializedMapping["left_study"] = stackLen
		return deserializedMapping
	}
	return nil
}

func reviewBtnError(c *gin.Context) {
	reviewBtnErrorResult := btnError("review", c)
	if reviewBtnErrorResult == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, reviewBtnErrorResult)
}

func studyBtnError(c *gin.Context) {
	studyBtnErrorResult := btnError("study", c)
	if studyBtnErrorResult == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, studyBtnErrorResult)
}

func wordNo(wordNoType string, c *gin.Context) map[string]interface{} {
	id, _ := strconv.Atoi(c.Param("id"))
	rdb := global.RedisDB
	currentUserId, exist := c.Get("userID")
	if !exist {
		return nil
	}
	userId := currentUserId.(string)
	stackUserKey := wordNoType + ":" + userId
	wordUserKey := stackUserKey + ":" + strconv.Itoa(id)
	result, _ := rdb.Exists(wordUserKey).Result()
	if result != 0 {
		stackLen, _ := rdb.LLen(stackUserKey).Result()
		topWordId, _ := rdb.LPop(stackUserKey).Result()
		if stackLen > 1 {
			rand.Seed(time.Now().UnixNano())
			randomIndex := rand.Intn(int(stackLen))
			// 获取随机索引位置的元素
			pivot, _ := rdb.LIndex(stackUserKey, int64(randomIndex)).Result()
			// 向List的随机位置插入数据
			rdb.LInsertBefore(stackUserKey, pivot, topWordId)
		} else {
			rdb.RPush(stackUserKey, topWordId)
		}
		return map[string]interface{}{
			"status": "success",
		}
	}
	return nil
}

func studyWordNo(c *gin.Context) {
	studyWordNoResult := wordNo("study", c)
	if studyWordNoResult == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, studyWordNoResult)
}

func reviewWordNo(c *gin.Context) {
	reviewWordNoResult := wordNo("review", c)
	if reviewWordNoResult == nil {
		c.JSON(200, gin.H{
			"status": "error",
		})
		return
	}
	c.JSONP(200, reviewWordNoResult)
}

func wordList(c *gin.Context) {
	db := global.MySQLDB
	var words []global.Word
	db.Find(&words)
	// Create a list to hold the data
	wList := make([]map[string]interface{}, 0)

	// Iterate over words to create the desired structure
	for _, word := range words {
		mList := make([]map[string]string, 0)
		for _, meaning := range word.Meanings {
			mList = append(mList, map[string]string{
				"part_of_speech": meaning.PartOfSpeech,
				"definition":     meaning.Definition,
			})
		}
		wList = append(wList, map[string]interface{}{
			"wid":      word.ID,
			"word":     word.Word,
			"meanings": mList,
			"num":      len(mList),
		})
	}
	c.JSONP(200, wList)
}

func SetupWordRouter(wordGroup *gin.RouterGroup) {
	wordGroup.GET("/", wordIndex)
	wordGroup.GET("/study/:id", wordStudyDetail)
	wordGroup.GET("/review/:id", wordReviewDetail)
	wordGroup.POST("/study/yes/:id", studyWordYes)
	wordGroup.POST("/study/no/:id", studyWordNo)
	wordGroup.POST("/review/yes/:id", reviewWordYes)
	wordGroup.POST("/review/no/:id", reviewWordNo)
	wordGroup.Match([]string{"GET", "POST"}, "/study/btn_next/:id", studyBtnNext)
	wordGroup.Match([]string{"GET", "POST"}, "/study/btn_error/:id", studyBtnError)
	wordGroup.Match([]string{"GET", "POST"}, "/review/btn_next/:id", reviewBtnNext)
	wordGroup.Match([]string{"GET", "POST"}, "/review/btn_error/:id", reviewBtnError)
	wordGroup.POST("/wordList/:id", wordList)
}
