package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
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
	// PureJSON avoids encoding '&' as \u0026 in URL fields (e.g. SAS query strings).
	c.Render(status, render.PureJSON{Data: body})
}

func respondMessage(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": true, "message": message, "timestamp": serverTimestamp()})
}
