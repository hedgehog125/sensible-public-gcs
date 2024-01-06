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

		if capTotalReqCount(ctx, state, env) {
			return
		}

		gcpRequestMade := false
		defer func() {
			go undoTotalReqCountIfNotSent(gcpRequestMade, state)
		}()

		if capTotalEgress(constants.MIN_REQUEST_EGRESS, 0, ctx, state, env) {
			return
		}

		objectPath := ctx.Param("path")[1:]

		user, userChan := getUser(ip, state, env)
		// The lock is only released when the response body starts to be sent which isn't super efficient, but good enough for this
		responseSent := false
		defer func() {
			go func() {
				if !responseSent { // If a response has been sent, the user will already have been unlocked
					go func() { *userChan <- user }()
				}
			}()
		}()

		UserTick(user, time.Now(), env)
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
		if capTotalEgress(reqEgress, constants.MIN_REQUEST_EGRESS, ctx, state, env) {
			_ = res.Body.Close()
			reqEgress = constants.MIN_REQUEST_EGRESS // So the defer subtracts the right value when updating the totals using "written"
			return
		}

		if secondCapUserEgress(reqEgress, user, ctx, env) {
			_ = res.Body.Close()
			return
		}

		user.EgressUsed += reqEgress
		go func() { *userChan <- user }()
		responseSent = true

		written, _ = io.Copy(ctx.Writer, res.Body)
		// Then correctEgressAfter runs as a defer
	})
}

// Returns true if it's sent a 503
// Also increases state.MonthlyRequestCount
func capTotalReqCount(ctx *gin.Context, state *intertypes.State, env *intertypes.Env) bool {
	reqCount := <-*state.MonthlyRequestCount
	newReqCount := reqCount + 1
	if newReqCount >= env.MAX_TOTAL_REQUESTS {
		go func() { *state.MonthlyRequestCount <- reqCount }()
		util.Send503(ctx) // TODO: move to outer function
		return true
	}
	reqCount = newReqCount
	go func() { *state.MonthlyRequestCount <- reqCount }()

	return false
}
func undoTotalReqCountIfNotSent(gcpRequestMade bool, state *intertypes.State) {
	if !gcpRequestMade {
		reqCount := <-*state.MonthlyRequestCount
		reqCount--
		go func() { *state.MonthlyRequestCount <- reqCount }()
	}
}

// Returns true if it's sent a 503
//
// Also increases state.ProvisionalAdditionalEgress
func capTotalEgress(
	reqEgress int64, formerProvReqEgress int64,
	ctx *gin.Context, state *intertypes.State, env *intertypes.Env,
) bool {
	provEgress := <-*state.ProvisionalAdditionalEgress
	cautiousTotalEgress := state.MeasuredEgress + provEgress
	remainingCautiousTotalEgress := env.MAX_TOTAL_EGRESS - cautiousTotalEgress

	// Minus formerProvReqEgress because the total egress was temporarily increased by that earlier
	if remainingCautiousTotalEgress < reqEgress-formerProvReqEgress {
		go func() { *state.ProvisionalAdditionalEgress <- provEgress }()
		util.Send503(ctx) // TODO: move to outer function
		return true
	}
	provEgress -= formerProvReqEgress
	provEgress += reqEgress
	go func() { *state.ProvisionalAdditionalEgress <- provEgress }()

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

// Note: this creates the user if it doesn't exist
func getUser(ip string, state *intertypes.State, env *intertypes.Env) (*intertypes.User, *chan *intertypes.User) {
	userChan, exists := state.Users[ip]
	var user *intertypes.User
	if exists {
		user = <-*userChan
	} else {
		fmt.Printf("New user: %v\n", ip)
		user = &intertypes.User{
			ResetAt: time.Now().Add(env.USER_RESET_TIME),
		}
		userChan = util.Pointer(make(chan *intertypes.User))

		go func() { *userChan <- user }()
		state.Users[ip] = userChan
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
			provEgress := <-*state.ProvisionalAdditionalEgress
			provEgress -= constants.MIN_REQUEST_EGRESS
			go func() { *state.ProvisionalAdditionalEgress <- provEgress }()
		}()
		return true
	}
	return false
}

// Returns true if it's sent a 429
func secondCapUserEgress(
	reqEgress int64, user *intertypes.User,
	ctx *gin.Context, env *intertypes.Env,
) bool { // TODO: does this need to be a separate function?
	if user.EgressUsed+reqEgress > env.DAILY_EGRESS_PER_USER {
		util.Send429(ctx, user)
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
		userChan, exists := state.Users[ip]
		if exists {
			user := <-*userChan
			// Unlike the total, MIN_REQUEST_EGRESS is never added to the user egress so it doesn't need refunding
			user.EgressUsed -= reqEgress
			user.EgressUsed += actualReqEgress
			go func() { *userChan <- user }()
		}
	}

	// Update provisional egress
	provEgress := <-*state.ProvisionalAdditionalEgress
	provEgress -= reqEgress
	provEgress += actualReqEgress
	go func() { *state.ProvisionalAdditionalEgress <- provEgress }()

	time.Sleep(env.GCP_EGRESS_LATENCY)

	provEgress = <-*state.ProvisionalAdditionalEgress
	provEgress -= actualReqEgress
	go func() { *state.ProvisionalAdditionalEgress <- provEgress }()
}

// Returns true if the user can now be forgotten
func UserTick(user *intertypes.User, now time.Time, env *intertypes.Env) bool {
	if now.After(user.ResetAt) {
		user.EgressUsed = 0
		user.ResetAt = now.Add(env.USER_RESET_TIME)
	}
	return user.EgressUsed == 0
}
