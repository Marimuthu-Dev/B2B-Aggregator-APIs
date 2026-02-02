package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
)

func serverTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func respondData(c *gin.Context, status int, data interface{}, message string, extra gin.H) {
	if message == "" {
		message = "Success"
	}
	body := gin.H{"success": true, "data": data, "message": message, "timestamp": serverTimestamp()}
	for key, value := range extra {
		body[key] = value
	}
	c.JSON(status, body)
}

func respondMessage(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": true, "message": message, "timestamp": serverTimestamp()})
}
