package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Khởi tạo router
	r := gin.Default()

	// Định nghĩa một route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Xin chào từ Gin!",
		})
	})

	// Chạy server trên cổng 8080
	r.Run(":8080")
}
