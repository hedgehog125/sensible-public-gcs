package endpoints

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/constants"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/util"
)

func Object(r *gin.Engine, client intertypes.GCPClient, state *intertypes.State, env *intertypes.Env) {
	r.GET("/v1/object/*path", func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		if ip == "" {
			ctx.Data(http.StatusUnauthorized, "text/plain", []byte("Couldn't find IP address"))
			return
		}

		if capTotalReqCount(state, env) {
			util.Send503(ctx)
			return
		}

		gcpRequestMade := false
		defer func() {
			go undoTotalReqCountIfNotSent(gcpRequestMade, state)
		}()

		if capTotalEgress(constants.MIN_REQUEST_EGRESS, 0, state, env) {
			util.Send503(ctx)
			return
		}

		objectPath := ctx.Param("path")[1:]

		user, userChan := getUser(ip, true, state, env)
		// The lock is only released when the response body starts to be sent which isn't super efficient, but good enough for this
		responseSent := false
		defer func() {
			go func() {
				if !responseSent { // If a response has been sent, the user will already have been unlocked
					go func() { userChan <- user }()
				}
			}()
		}()

		UserTick(user, time.Now().UTC(), env)
		if initialCapUserEgress(user, ctx, state, env) {
			return
		}

		reqEgress := constants.MIN_REQUEST_EGRESS
		written := int64(0)
		defer func() {
			go correctEgressAfter(responseSent, written, reqEgress, ip, state, env)
		}()

		gcpRequestMade = true
		res, didErr := client.FetchObject(objectPath, ctx)
		if didErr {
			util.Send500(ctx)
			return
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			_ = res.Body.Close()
			if res.StatusCode == 404 {
				ctx.Status(res.StatusCode)
			} else {
				util.Send500(ctx)
				fmt.Printf("warning: response from GCP had %v status\n", res.StatusCode)

				// If the GCP credentials were invalid, creating the signed URL would have failed instead of this
				if res.StatusCode == 400 {
					fmt.Printf("Is this UTC time correct to within 15 seconds?\n%v\n", time.Now().UTC().String())
				}
				fmt.Println("")
			}
			return
		}
		copyStatusAndHeaders(res, ctx)

		contentLength, didErr := parseContentLength(res)
		if didErr {
			_ = res.Body.Close()
			util.Send500(ctx)
			return
		}

		reqEgress = max(contentLength+constants.ASSUMED_OVERHEAD, constants.MIN_REQUEST_EGRESS)
		if capTotalEgress(reqEgress, constants.MIN_REQUEST_EGRESS, state, env) {
			_ = res.Body.Close()
			reqEgress = constants.MIN_REQUEST_EGRESS // So the defer subtracts the right value when updating the totals using "written"
			util.Send503(ctx)
			return
		}

		{
			newEgressUsed := user.EgressUsed + reqEgress
			if newEgressUsed > env.DAILY_EGRESS_PER_USER {
				user.EgressUsed += constants.MIN_REQUEST_EGRESS

				util.Send429(ctx, user)
				_ = res.Body.Close()
				return
			}

			user.EgressUsed = newEgressUsed
		}
		go func() { userChan <- user }()
		responseSent = true

		written, _ = io.Copy(ctx.Writer, res.Body)
		// Then correctEgressAfter runs as a defer
	})
}

// Returns true if the request should be blocked
// Also increases state.MonthlyRequestCount
func capTotalReqCount(state *intertypes.State, env *intertypes.Env) bool {
	reqCount := <-state.MonthlyRequestCount
	newReqCount := reqCount + 1
	if newReqCount > env.MAX_TOTAL_REQUESTS {
		go func() { state.MonthlyRequestCount <- reqCount }()
		return true
	}
	reqCount = newReqCount
	go func() { state.MonthlyRequestCount <- reqCount }()

	return false
}
func undoTotalReqCountIfNotSent(gcpRequestMade bool, state *intertypes.State) {
	if !gcpRequestMade {
		reqCount := <-state.MonthlyRequestCount
		reqCount--
		go func() { state.MonthlyRequestCount <- reqCount }()
	}
}

// Returns true if the request should be blocked
//
// Also increases state.ProvisionalAdditionalEgress
func capTotalEgress(
	reqEgress int64, formerProvReqEgress int64,
	state *intertypes.State, env *intertypes.Env,
) bool {
	provEgress := <-state.ProvisionalAdditionalEgress
	cautiousTotalEgress := state.MeasuredEgress.SimpleRead() + provEgress
	remainingCautiousTotalEgress := env.MAX_TOTAL_EGRESS - cautiousTotalEgress

	// Minus formerProvReqEgress because the total egress was temporarily increased by that earlier
	if remainingCautiousTotalEgress < reqEgress-formerProvReqEgress {
		go func() { state.ProvisionalAdditionalEgress <- provEgress }()
		return true
	}
	provEgress -= formerProvReqEgress
	provEgress += reqEgress
	go func() { state.ProvisionalAdditionalEgress <- provEgress }()

	return false
}

// 2nd return value is true if an error occurred
//
// Note: this doesn't send the error response itself
func parseContentLength(res *http.Response) (int64, bool) {
	contentLengthStr := res.Header.Get("content-length")
	if contentLengthStr == "" {
		return 0, true
	}
	contentLength, err := strconv.ParseInt(contentLengthStr, 10, 0)
	if err != nil {
		return 0, true
	}
	return contentLength, false
}

// Note: if createIfDoesntExist is false, the returned channel will be nil as opposed to pointing to the nil user
func getUser(
	ip string, createIfDoesntExist bool,
	state *intertypes.State, env *intertypes.Env,
) (*intertypes.User, chan *intertypes.User) {
	userChan, exists := state.Users.Load(ip)
	var user *intertypes.User
	if exists {
		user = <-userChan
		// The user could have been deleted from the map while we were getting the lock, in which case it'll have been set to nil
		if user == nil {
			// We still need to put it back though in case there's a queue, otherwise those goroutines will hang forever
			go func() { userChan <- nil }()
			exists = false
		}
	}

	if !exists {
		if !createIfDoesntExist {
			return nil, nil
		}
		if !env.DISABLE_REQUEST_LOGS {
			fmt.Printf("new user: %v\n", ip)
		}
		user = &intertypes.User{
			ResetAt: time.Now().UTC().Add(env.USER_RESET_TIME),
		}
		userChan = make(chan *intertypes.User)
		// user will be put into the channel once the calling function is done with it

		state.Users.Store(ip, userChan)
	}

	return user, userChan
}

// Returns true if it's sent a 429
//
// Based on MIN_REQUEST_EGRESS rather than an actual number at this stage
func initialCapUserEgress(
	user *intertypes.User, ctx *gin.Context,
	state *intertypes.State, env *intertypes.Env,
) bool {
	remaining := env.DAILY_EGRESS_PER_USER - user.EgressUsed
	if remaining < constants.MIN_REQUEST_EGRESS {
		util.Send429(ctx, user)
		// Refund the total egress now rather than waiting for the 3 minutes
		go func() {
			provEgress := <-state.ProvisionalAdditionalEgress
			provEgress -= constants.MIN_REQUEST_EGRESS
			go func() { state.ProvisionalAdditionalEgress <- provEgress }()
		}()
		return true
	}
	return false
}

func copyStatusAndHeaders(res *http.Response, ctx *gin.Context) {
	for _, headerName := range constants.PROXIED_HEADERS {
		ctx.Header(headerName, res.Header.Get(headerName))
	}
	ctx.Status(res.StatusCode)
}

// Runs after the whole response has been sent and updates the numbers with how much was actually sent
func correctEgressAfter(
	responseSent bool, written int64,
	reqEgress int64, ip string,
	state *intertypes.State, env *intertypes.Env,
) {
	actualReqEgress := max(written+constants.ASSUMED_OVERHEAD, constants.MIN_REQUEST_EGRESS)

	if responseSent {
		userChan, exists := state.Users.Load(ip)
		if exists {
			user := <-userChan
			if user != nil {
				// Unlike the total, MIN_REQUEST_EGRESS is never added to the user egress so it doesn't need refunding
				user.EgressUsed -= reqEgress
				user.EgressUsed += actualReqEgress
			}
			go func() { userChan <- user }()
		}
	}

	// Update provisional egress
	provEgress := <-state.ProvisionalAdditionalEgress
	provEgress -= reqEgress
	provEgress += actualReqEgress
	go func() { state.ProvisionalAdditionalEgress <- provEgress }()

	time.Sleep(env.GCP_EGRESS_LATENCY)

	provEgress = <-state.ProvisionalAdditionalEgress
	provEgress -= actualReqEgress
	go func() { state.ProvisionalAdditionalEgress <- provEgress }()
}

// Returns true if the user can now be forgotten
func UserTick(user *intertypes.User, now time.Time, env *intertypes.Env) bool {
	if now.After(user.ResetAt) {
		user.EgressUsed = 0
		user.ResetAt = now.Add(env.USER_RESET_TIME)
	}
	return user.EgressUsed == 0
}
