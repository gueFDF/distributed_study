package main

import (
	"gin"
	"net/http"
)

func main() {
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
	})
	r.GET("/hello", func(c *gin.Context) {
		// expect /hello?name=geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	r.GET("/hello/:name", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello %s, you're at %s\n", c.Param("name"), c.Path)
	})

	r.GET("/assert/*filepath", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"filepath": c.Param("filepath")})
	})
	r.Run(":9999")
}
