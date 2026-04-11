package handlers

import (
	"errors"
	"net/http"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func respondError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	if appErr := apperrors.From(err); appErr != nil {
		status := http.StatusInternalServerError
		switch appErr.Kind {
		case apperrors.KindBadRequest:
			status = http.StatusBadRequest
		case apperrors.KindUnauthorized:
			status = http.StatusUnauthorized
		case apperrors.KindForbidden:
			status = http.StatusForbidden
		case apperrors.KindNotFound:
			status = http.StatusNotFound
		case apperrors.KindConflict:
			status = http.StatusConflict
		case apperrors.KindInternal:
			status = http.StatusInternalServerError
		}

		message := appErr.Message
		if message == "" {
			message = http.StatusText(status)
		}

		c.JSON(status, gin.H{"Success": false, "Message": message, "Timestamp": serverTimestamp()})
		return
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"Success": false, "Message": "Resource not found", "Timestamp": serverTimestamp()})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"Success": false, "Message": err.Error(), "Timestamp": serverTimestamp()})
}
