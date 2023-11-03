package endpoints

import (
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
)

func Health(r *gin.Engine) {
	r.GET("/health", func(ctx *gin.Context) {
		// There's no asynchronous setup, so just send a 200
		ctx.Data(200, "text/plain", []byte(""))
	})
}

func Redirect(r *gin.Engine, bucket *storage.BucketHandle) {
	r.GET("/v1/redirect/*path", func(ctx *gin.Context) {
		objURL, err := bucket.SignedURL(
			ctx.Param("path")[1:],
			&storage.SignedURLOptions{
				Method:  "GET",
				Expires: time.Now().Add(10 * time.Second),
			},
		)
		if err != nil {
			ctx.Data(500, "text/plain", []byte(""))
			return
		}

		// TODO: allow caching until a couple seconds before the URL expires
		ctx.Header("cache-control", "no-store")
		ctx.Redirect(307, objURL)
	})
}
