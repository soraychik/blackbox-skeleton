package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Устанавливаем режим Gin
	gin.SetMode(gin.ReleaseMode)

	// Создаем роутер
	router := gin.Default()

	// Главная страница
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "BlackBox API Web is running... (stub)")
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	log.Println("API Web server starting on :8080")

	// Запускаем сервер
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
