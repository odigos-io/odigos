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

func returnErrors(c *gin.Context, errors []error) {
	errorsText := make([]string, len(errors))
	for i, err := range errors {
		errorsText[i] = err.Error()
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"messages": errorsText,
	})
}
