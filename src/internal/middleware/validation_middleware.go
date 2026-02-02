package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		c.Error(err).SetType(gin.ErrorTypeBind)
		c.Abort()
		return false
	}
	return true
}

func BindQuery(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		c.Error(err).SetType(gin.ErrorTypeBind)
		c.Abort()
		return false
	}
	return true
}

func BindUri(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindUri(obj); err != nil {
		c.Error(err).SetType(gin.ErrorTypeBind)
		c.Abort()
		return false
	}
	return true
}

func ValidationErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}

		for _, err := range c.Errors {
			if err.Type != gin.ErrorTypeBind {
				continue
			}

			if ve, ok := err.Err.(validator.ValidationErrors); ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"success":   false,
					"message":   "Validation failed",
					"errors":    formatValidationErrors(ve),
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
				return
			}

			var typeErr *json.UnmarshalTypeError
			if errors.As(err.Err, &typeErr) {
				c.JSON(http.StatusBadRequest, gin.H{
					"success":   false,
					"message":   "Invalid type for field " + typeErr.Field,
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
				return
			}

			c.JSON(http.StatusBadRequest, gin.H{
				"success":   false,
				"message":   err.Err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
	}
}

func formatValidationErrors(ve validator.ValidationErrors) []string {
	messages := make([]string, 0, len(ve))
	for _, fe := range ve {
		field := fe.Field()
		tag := fe.Tag()
		if fe.Param() != "" {
			tag = tag + "=" + fe.Param()
		}
		messages = append(messages, field+": "+strings.ToLower(tag))
	}
	return messages
}
