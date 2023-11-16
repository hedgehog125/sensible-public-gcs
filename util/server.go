package util

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func Send500(ctx *gin.Context) {
	ctx.Data(500, "text/plain", []byte(""))
}

func Send429(ctx *gin.Context, user *intertypes.User) {
	secondsUntilReset := user.ResetAt - time.Now().Unix()
	ctx.Header("retry-after", strconv.FormatInt(secondsUntilReset, 10))
	ctx.Data(http.StatusTooManyRequests, "text/plain", []byte(""))
}
