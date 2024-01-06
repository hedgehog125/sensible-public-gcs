package endpoints

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

type RemainingEgressResponse struct {
	Used      int64 `json:"used"`
	Remaining int64 `json:"remaining"`
}

func RemainingEgress(r *gin.Engine, state *intertypes.State, env *intertypes.Env) {
	r.GET("/v1/remaining/egress", func(ctx *gin.Context) {
		used := getUsed(ctx.ClientIP(), state, env)

		ctx.JSON(200, RemainingEgressResponse{
			Used:      used,
			Remaining: env.DAILY_EGRESS_PER_USER - used,
		})
	})
}
func getUsed(ip string, state *intertypes.State, env *intertypes.Env) int64 {
	user, userChan := getUser(ip, false, state, env)
	if user == nil {
		// No lock to release
		return 0
	}

	defer func() {
		go func() { *userChan <- user }()
	}()
	UserTick(user, time.Now().UTC(), env)
	return user.EgressUsed
}
