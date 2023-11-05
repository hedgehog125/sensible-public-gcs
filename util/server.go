package util

import "github.com/gin-gonic/gin"

func Send500(ctx *gin.Context) {
	ctx.Data(500, "text/plain", []byte(""))
}
