package subfns

import (
	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/endpoints"
)

func RegisterEndpoints(r *gin.Engine) {
	endpoints.Health(r)
}
