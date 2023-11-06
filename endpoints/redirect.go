package endpoints

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/constants"
	"github.com/hedgeghog125/sensible-public-gcs/util"
)

func Redirect(r *gin.Engine, bucket *storage.BucketHandle) {
	// It doesn't need to be cryptographically secure, it just needs to be long and incompressible
	longValue := util.RandomString(constants.RANDOM_QUERY_LENGTH)
	longQueryParams := url.Values{}
	longQueryParams.Add("rand", longValue)

	r.GET("/v1/redirect/*path", func(ctx *gin.Context) {
		if true {
			sendRedirect(ctx, bucket, &longQueryParams)
		} else {
			sendByProxy(ctx, bucket)
		}
	})
}
func sendRedirect(ctx *gin.Context, bucket *storage.BucketHandle, longQueryParams *url.Values) {
	objURL, err := createSignedURLForRequest(ctx, bucket, longQueryParams)
	if err != nil {
		util.Send500(ctx)
		return
	}

	// TODO: allow caching until a couple seconds before the URL expires
	ctx.Header("cache-control", "no-store")
	ctx.Redirect(307, objURL)
}
func sendByProxy(ctx *gin.Context, bucket *storage.BucketHandle) {
	objURL, err := createSignedURLForRequest(ctx, bucket, nil)
	if err != nil {
		util.Send500(ctx)
		return
	}

	req, err := http.NewRequestWithContext(ctx.Request.Context(), "GET", objURL, nil)
	if err != nil { // Invalid request?
		util.Send500(ctx)
		return
	}
	req.Header.Set("range", ctx.Request.Header.Get("range"))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		util.Send500(ctx)
		return
	}

	for _, headerName := range constants.PROXIED_HEADERS {
		ctx.Header(headerName, res.Header.Get(headerName))
	}
	ctx.Status(res.StatusCode)
	io.Copy(ctx.Writer, res.Body)
}
func createSignedURLForRequest(ctx *gin.Context, bucket *storage.BucketHandle, longQueryParams *url.Values) (string, error) {
	return bucket.SignedURL(
		ctx.Param("path")[1:],
		&storage.SignedURLOptions{
			Method:          "GET",
			Expires:         time.Now().Add(3 * time.Second),
			Scheme:          storage.SigningSchemeV4,
			QueryParameters: *longQueryParams,
		},
	)
}
