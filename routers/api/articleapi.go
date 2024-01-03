package api

import (
	"awesomeProject/global"
	"github.com/gin-gonic/gin"
)

func articleIndex(c *gin.Context) {
	c.String(200, "articleapi is ok")
}

func articleList(c *gin.Context) {
	db := global.MySQLDB
	var articles []map[string]interface{}
	db.Model(&global.Article{}).
		Select("id as aid, title,headpic,abstract").
		Order("RAND()").Limit(5).
		Scan(&articles)
	c.JSONP(200, articles)
	return
}

func articleDetail(c *gin.Context) {
	db := global.MySQLDB
	article := global.Article{}
	db.First(&article, c.Param("id"))
	c.JSON(200, gin.H{
		"headpic":     article.Headpic,
		"title":       article.Title,
		"body":        article.Body,
		"source":      article.Source,
		"source_href": article.SourceHref,
	})
}

func SetupArticleRouter(articleGroup *gin.RouterGroup) {
	articleGroup.GET("/", articleIndex)
	articleGroup.POST("/articleList", articleList)
	articleGroup.POST("/:id", articleDetail)
}
