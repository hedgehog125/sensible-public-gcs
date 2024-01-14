package subfns

import (
	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/endpoints"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func RegisterEndpoints(
	r *gin.Engine, client intertypes.GCPClient,
	state *intertypes.State, env *intertypes.Env,
) {
	endpoints.Health(r)

	endpoints.IP(r)
	if env.IS_PROXY_TEST {
		endpoints.Debug(r)
	} else {
		endpoints.Object(r, client, state, env)
		endpoints.RemainingEgress(r, state, env)
	}
}
