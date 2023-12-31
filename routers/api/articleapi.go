package api

import "github.com/gin-gonic/gin"

func articleIndex(c *gin.Context) {
	c.String(200, "articleapi is ok")
}

func SetupArticleRouter(wordGroup *gin.RouterGroup) {
	wordGroup.GET("/", articleIndex)
}
