package subfns

import (
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/endpoints"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func RegisterEndpoints(
	r *gin.Engine, bucket *storage.BucketHandle,
	state *intertypes.State, env *intertypes.Env,
) {
	endpoints.Health(r)
	endpoints.Object(r, bucket, state, env)
	endpoints.RemainingEgress(r, state, env)
}
