package endpoints

import "github.com/gin-gonic/gin"

func Health(r *gin.Engine) {
	r.GET("/health", func(ctx *gin.Context) {
		// There's no asynchronous setup, so just send a 200
		ctx.Data(200, "text/plain", []byte(""))
	})
}
