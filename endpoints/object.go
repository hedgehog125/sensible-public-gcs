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
		provEgress := <-*state.ProvisionalAdditionalEgress
		cautiousTotalEgress := state.MeasuredEgress + provEgress
		remainingCautiousTotalEgress := env.MAX_TOTAL_EGRESS - cautiousTotalEgress
		if remainingCautiousTotalEgress < constants.MIN_REQUEST_EGRESS {
			go func() { *state.ProvisionalAdditionalEgress <- provEgress }()
			util.Send503(ctx)
			return
		}
		provEgress += constants.MIN_REQUEST_EGRESS
		fmt.Printf("Request start, provEgress: %v | measuredEgress: %v\n", provEgress, state.MeasuredEgress)
		go func() { *state.ProvisionalAdditionalEgress <- provEgress }()

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
			userChan = util.Pointer[chan *intertypes.User](make(chan *intertypes.User))

			go func() { *userChan <- user }()
			state.Users[ip] = userChan
		}
		// The lock is only released when the response body starts to be sent which isn't super efficient, but good enough for this
		userLockReleased := false
		defer func() {
			go func() {
				if !userLockReleased {
					*userChan <- user
				}
			}()
		}()

		UserTick(user)
		remaining := env.DAILY_EGRESS_PER_USER - user.EgressUsed
		if remaining < constants.MIN_REQUEST_EGRESS {
			util.Send429(ctx, user)
			// Refund the total egress now rather than waiting for the 3 minutes
			go func() {
				provEgress = <-*state.ProvisionalAdditionalEgress
				provEgress -= constants.MIN_REQUEST_EGRESS
				*state.ProvisionalAdditionalEgress <- provEgress
			}()
			return
		}

		reqEgress := constants.MIN_REQUEST_EGRESS
		responseSent := false
		written := int64(0)
		defer func() {
			go func() {
				actualReqEgress := max(written+constants.ASSUMED_OVERHEAD, constants.MIN_REQUEST_EGRESS)

				if responseSent {
					userChan, exists = state.Users[ip]
					if exists {
						user = <-*userChan
						user.EgressUsed -= reqEgress
						user.EgressUsed += actualReqEgress
						go func() { *userChan <- user }()
					}
				}

				// Update provisional egress
				provEgress = <-*state.ProvisionalAdditionalEgress
				provEgress -= reqEgress
				provEgress += actualReqEgress
				fmt.Printf("Request finished, provEgress: %v | measuredEgress: %v\n", provEgress, state.MeasuredEgress)
				go func() { *state.ProvisionalAdditionalEgress <- provEgress }()

				time.Sleep(3 * time.Minute)

				provEgress = <-*state.ProvisionalAdditionalEgress
				provEgress -= actualReqEgress
				fmt.Printf("3 minutes after request finished, provEgress: %v | measuredEgress: %v\n", provEgress, state.MeasuredEgress)
				go func() { *state.ProvisionalAdditionalEgress <- provEgress }()
			}()
		}()

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

		reqEgress = max(contentLength+constants.ASSUMED_OVERHEAD, constants.MIN_REQUEST_EGRESS)
		provEgress = <-*state.ProvisionalAdditionalEgress
		cautiousTotalEgress = state.MeasuredEgress + provEgress
		if cautiousTotalEgress+reqEgress > env.MAX_TOTAL_EGRESS {
			_ = res.Body.Close()
			go func() { *state.ProvisionalAdditionalEgress <- provEgress }()
			util.Send503(ctx)
			return
		}

		provEgress -= constants.MIN_REQUEST_EGRESS
		provEgress += reqEgress
		fmt.Printf("Got request headers, provEgress: %v | measuredEgress: %v\n", provEgress, state.MeasuredEgress)
		go func() { *state.ProvisionalAdditionalEgress <- provEgress }()

		newUserEgress := user.EgressUsed + reqEgress
		if newUserEgress > env.DAILY_EGRESS_PER_USER {
			_ = res.Body.Close()
			util.Send429(ctx, user)
			return
		}

		user.EgressUsed = newUserEgress
		go func() { *userChan <- user }()
		userLockReleased = true
		responseSent = true

		written, _ = io.Copy(ctx.Writer, res.Body)
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
