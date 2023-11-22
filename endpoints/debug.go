package endpoints

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Debug(r *gin.Engine) {
	r.GET("/v1/headers", func(ctx *gin.Context) {
		headers := make(map[string]string)
		for key, values := range ctx.Request.Header {
			headers[key] = values[0]
		}

		ctx.JSON(http.StatusOK, gin.H{"headers": headers})
	})
}
