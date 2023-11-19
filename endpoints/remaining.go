package endpoints

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func RemainingEgress(r *gin.Engine, state *intertypes.State, env *intertypes.Env) {
	r.GET("/v1/remaining/egress", func(ctx *gin.Context) {
		used := getUsed(ctx.ClientIP(), state)

		ctx.JSON(200, gin.H{
			"used":      used,
			"remaining": env.DAILY_EGRESS_PER_USER - used,
		})
	})
}
func getUsed(ip string, state *intertypes.State) int64 {
	userChan, exists := state.Users[ip]
	if !exists {
		return 0
	}

	user := <-*userChan
	UserTick(user, time.Now().Unix())
	go func() { *userChan <- user }()
	return user.EgressUsed
}
