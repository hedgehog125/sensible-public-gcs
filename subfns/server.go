package subfns

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func CreateServer() *gin.Engine {
	r := gin.Default()
	r.SetTrustedProxies(nil)

	return r
}
func AddMiddleware(r *gin.Engine, env *intertypes.Env) {
	r.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowOrigins: env.CORS_ALLOWED_ORIGINS,
		MaxAge:       5 * time.Minute,
	}))
}
func StartServer(r *gin.Engine, env *intertypes.Env) {
	r.Run(fmt.Sprintf(":%v", env.PORT))
}
