package endpoints

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func returnError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"message": err.Error(),
	})
}
