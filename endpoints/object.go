package endpoints

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/constants"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/util"
)

func Object(r *gin.Engine, bucket *storage.BucketHandle, state *intertypes.State, env *intertypes.Env) {
	r.GET("/v1/object/*path", func(ctx *gin.Context) {
		objectPath := ctx.Param("path")[1:]
		ip := ctx.ClientIP()

		userChan, exists := state.Users[ip]
		var user *intertypes.User
		if exists {
			user = <-*userChan
		} else {
			user = &intertypes.User{
				ResetAt: time.Now().Add(24 * time.Hour).Unix(),
			}
			newUserChan := make(chan *intertypes.User)
			userChan = &newUserChan

			go func() { *userChan <- user }()
			state.Users[ip] = userChan
		}
		// The lock is only released when the response body starts to be sent which isn't super efficient, but good enough for this
		lockReleased := false
		defer func() {
			go func() {
				if lockReleased {
					return
				}
				*userChan <- user
			}()
		}()

		UserTick(user)
		fmt.Printf("Egress: %v\n", user.EgressUsed)
		remaining := env.DAILY_EGRESS_PER_USER - user.EgressUsed
		if remaining < constants.MIN_REQUEST_EGRESS {
			util.Send429(ctx, user)
			return
		}

		objURL, err := bucket.SignedURL(
			objectPath,
			&storage.SignedURLOptions{
				Method:  "GET",
				Expires: time.Now().Add(3 * time.Second),
				Scheme:  storage.SigningSchemeV4,
			},
		)
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

		handleError := func() {
			_ = res.Body.Close()
			util.Send500(ctx)
		}
		contentLengthStr := res.Header.Get("content-length")
		if contentLengthStr == "" {
			handleError()
			return
		}
		contentLength, err := strconv.ParseInt(contentLengthStr, 10, 0)
		if err != nil {
			handleError()
			return
		}

		reqEgress := max(contentLength+constants.ASSUMED_OVERHEAD, constants.MIN_REQUEST_EGRESS)
		newUserEgress := user.EgressUsed + reqEgress
		if newUserEgress > env.DAILY_EGRESS_PER_USER {
			_ = res.Body.Close()
			util.Send429(ctx, user)
			return
		}

		user.EgressUsed = newUserEgress
		go func() { *userChan <- user }()
		lockReleased = true

		written, _ := io.Copy(ctx.Writer, res.Body)

		go func() {
			actualReqEgress := max(written+constants.ASSUMED_OVERHEAD, constants.MIN_REQUEST_EGRESS)

			userChan, exists = state.Users[ip]
			if !exists {
				return
			}
			user = <-*userChan
			fmt.Printf("Sent %v%%\n", (actualReqEgress*100)/reqEgress)
			user.EgressUsed -= reqEgress
			user.EgressUsed += actualReqEgress
			*userChan <- user
		}()
	})
}

// Returns true if the user can now be forgotten
func UserTick(user *intertypes.User) bool {
	if time.Now().Unix() >= user.ResetAt {
		user.EgressUsed = 0
		return true
	}
	return false
}
