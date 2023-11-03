package subfns

import (
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/endpoints"
)

func RegisterEndpoints(r *gin.Engine, bucket *storage.BucketHandle) {
	endpoints.Health(r)
	endpoints.Redirect(r, bucket)
}
