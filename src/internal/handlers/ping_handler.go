package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	pingTimeLayout = "02-01-2006 15:04:05"
	istZone        = "Asia/Kolkata"
)

// Ping returns environment, server and IST timestamps, and build/commit info.
// Environment from env ENVIRONMENT; Last Build Pushed and Latest commit from env (dev updates these).
func Ping(c *gin.Context) {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "dev"
	}

	now := time.Now()
	currentTS := now.Format(pingTimeLayout)

	istLoc, err := time.LoadLocation(istZone)
	if err != nil {
		istLoc = time.FixedZone("IST", 5*60*60+30*60) // UTC+5:30
	}
	istTS := now.In(istLoc).Format(pingTimeLayout)

	c.JSON(http.StatusOK, gin.H{
		"Environment":       env,
		"Current TimeStamp": currentTS,
		"IST TimeStamp":     istTS,
		"Last Build Pushed": "02-Mar-2026 02:15:00",
		"Latest commit":     "Bug fixes",
	})
}
