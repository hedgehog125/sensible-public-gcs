package util

import (
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func SendBlankStatus(code int, ctx *gin.Context) {
	ctx.Data(code, "text/plain", []byte(""))
}
func Send500(ctx *gin.Context) {
	SendBlankStatus(500, ctx)
}
func Send503(ctx *gin.Context) {
	SendBlankStatus(503, ctx)
}
func Send429(ctx *gin.Context, user *intertypes.User) {
	timeUntilReset := time.Until(user.ResetAt)
	secondsUntilReset := int64(math.Ceil(timeUntilReset.Seconds()))
	ctx.Header("retry-after", strconv.FormatInt(secondsUntilReset, 10))
	SendBlankStatus(429, ctx)
}
