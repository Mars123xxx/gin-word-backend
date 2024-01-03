package api

import (
	"awesomeProject/global"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
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
		sentences    []map[string]interface{}
		meanings     []map[string]interface{}
		collocations []map[string]interface{}
		relatives    []map[string]interface{}
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

func wordStudyDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	rdb := global.RedisDB
	currentUserId, exist := c.Get("userID")
	userId := currentUserId.(string)
	if exist {
		result, err := rdb.Exists("study:" + userId + ":" + strconv.Itoa(id)).Result()
		if err != nil {
			println("redis错误")
			return
		}
		if result != 0 {
			//这里返回一个map字典map[string]string
			data, _ := rdb.HGetAll("study:" + userId + ":" + strconv.Itoa(id)).Result()
			// 创建一个新的映射，用于存储反序列化后的JSON字符串,批量反序列化
			deserializedMapping := make(map[string]interface{})

			// 遍历映射，并序列化每个值
			for k, v := range data {
				var obj interface{}
				err := json.Unmarshal([]byte(v), &obj)
				if err != nil {
					return
				}
				deserializedMapping[k] = obj
			}
			c.JSONP(200, deserializedMapping)
			return
		}
		wordDetail, err := GetWordDetailByID(id)
		if err != nil {
			c.JSON(200, gin.H{
				"status": "error",
			})
			return
		}
		c.JSON(200, wordDetail)
		serializedMapping := make(map[string]interface{})
		for k, v := range wordDetail {
			// 遍历映射，并序列化每个值
			obj, err := json.Marshal(v)
			if err != nil {
				return
			}
			serializedMapping[k] = obj
		}
		rdb.HMSet("study:"+userId+":"+strconv.Itoa(id), serializedMapping)
		return
	} else {
		c.JSON(500, gin.H{
			"msg": "内部错误",
		})
	}
}

func wordReviewDetail(c *gin.Context) {
	c.JSON(200, gin.H{
		"msg": "hello",
	})
	//TODO
	//复习部分
}

func SetupWordRouter(wordGroup *gin.RouterGroup) {
	wordGroup.GET("/", wordIndex)
	wordGroup.GET("/study/:id", wordStudyDetail)
	wordGroup.GET("/review/:id", wordReviewDetail)
}
