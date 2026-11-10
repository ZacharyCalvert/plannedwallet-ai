package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "world")
	})

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
