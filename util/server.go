package util

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func Send500(ctx *gin.Context) {
	ctx.Data(500, "text/plain", []byte(""))
}
func Send503(ctx *gin.Context) {
	ctx.Data(503, "text/plain", []byte(""))
}
func Send429(ctx *gin.Context, user *intertypes.User) {
	timeUntilReset := time.Until(user.ResetAt)
	secondsUntilReset := int64(math.Ceil(timeUntilReset.Seconds()))
	ctx.Header("retry-after", strconv.FormatInt(secondsUntilReset, 10))
	ctx.Data(http.StatusTooManyRequests, "text/plain", []byte(""))
}
