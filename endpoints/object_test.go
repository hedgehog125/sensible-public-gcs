package endpoints_test

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hedgeghog125/sensible-public-gcs/constants"
	"github.com/hedgeghog125/sensible-public-gcs/endpoints"
	"github.com/hedgeghog125/sensible-public-gcs/test"
	"github.com/stretchr/testify/assert"
)

func TestBasicObject(t *testing.T) {
	r, _, env := test.InitProgram(nil)

	w := test.Fetch("GET", "/v1/object/foo.bar", nil, r, env)
	assert.Equal(t, 200, w.Code)
}
func Test404Object(t *testing.T) {
	r, _, env := test.InitProgram(nil)

	w := test.Fetch("GET", "/v1/object/404", nil, r, env)
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "", w.Body.String())
}

// The objects are all 1 byte so the minimum is used
//
// Note: 'Monthly' resets during this test are fine
func TestContinuousRequestsOfMinSize(t *testing.T) {
	testContiniousRequests(
		1, // 1 byte objects
		true,
		t,
	)
}

// Note: 'Monthly' resets during this test are fine
func TestContinuousRequestsOfIndivisibleSize(t *testing.T) {
	const reqSize = 7_500_000 // 7.5MB objects
	assert.Greater(t, int64(reqSize), constants.MIN_REQUEST_EGRESS)

	testContiniousRequests(
		reqSize,
		false,
		t,
	)
}
func testContiniousRequests(
	reqSize int, shouldBeDivisible bool,
	t *testing.T,
) {
	r, state, env := test.InitProgram(&test.Config{
		RandomContentLength: reqSize,
	})

	reqSizePlusOverhead := int64(reqSize) + constants.ASSUMED_OVERHEAD
	effectiveSize := max(reqSizePlusOverhead, constants.MIN_REQUEST_EGRESS)
	assert.Greater(t, env.DAILY_EGRESS_PER_USER, effectiveSize) // Not GreaterOrEqual so it needs to be at least 2 requests
	isDivisible := env.DAILY_EGRESS_PER_USER%effectiveSize == 0
	if isDivisible != shouldBeDivisible {
		possibleWord := ""
		if shouldBeDivisible {
			possibleWord = " not"
		}

		t.Fatalf(
			"env.DAILY_EGRESS_PER_USER (%v) is%v divisible by the effective request size (%v)",
			env.DAILY_EGRESS_PER_USER,
			possibleWord,
			effectiveSize,
		)
	}

	totalReqCount := int64(0)
	totalBodyEgress := int64(0)
	checkTotalReqCount := func() {
		assert.Equal(t, totalReqCount, test.ReadChannel(state.MonthlyRequestCount))
	}

	runTests := func() time.Duration {
		startTime := time.Now().UTC()
		var total int64
		for total = int64(0); total+effectiveSize <= env.DAILY_EGRESS_PER_USER; {
			w := test.Fetch("GET", "/v1/object/foo.bar", nil, r, env)
			assert.Equal(t, 200, w.Code)
			_ = w.Body.String()

			total += effectiveSize
			totalBodyEgress += reqSizePlusOverhead
			totalReqCount++
			checkTotalReqCount()
		}

		checkRemainingUserEgress := func() {
			w := test.Fetch("GET", "/v1/remaining/egress", nil, r, env)
			assert.Equal(t, 200, w.Code)
			jsonRes := endpoints.RemainingEgressResponse{}
			err := json.NewDecoder(w.Body).Decode(&jsonRes)
			if err != nil {
				t.Fatalf(err.Error())
				return
			}
			assert.Equal(t, env.DAILY_EGRESS_PER_USER-total, jsonRes.Remaining)
			assert.Equal(t, total, jsonRes.Used)
		}
		checkRemainingUserEgress()
		checkTotalReqCount()

		if time.Since(startTime) >= env.USER_RESET_TIME-2 {
			t.Fatal("couldn't reach daily limit before it was reset. Is your computer busy?")
		}

		expect429 := func() {
			w := test.Fetch("GET", "/v1/object/foo.bar", nil, r, env)
			assert.Equal(t, 429, w.Code)
			_ = w.Body.String()

			time.Sleep(2 * time.Millisecond) // Give it 2ms to undo the increment

			remaining := env.DAILY_EGRESS_PER_USER - total
			if remaining >= constants.MIN_REQUEST_EGRESS {
				// The server will send a request to GCP but won't send it to the client as it'll realise it's too big
				total += constants.MIN_REQUEST_EGRESS // It still counts as user egress usage
				totalReqCount++
				totalBodyEgress += reqSizePlusOverhead // The mock GCP client doesn't handle cancelled requests
				checkRemainingUserEgress()
			}

			checkTotalReqCount() // Make sure it hasn't increased
		}
		expect429()
		time.Sleep(3 * time.Millisecond)
		expect429()

		return time.Since(startTime)
	}

	elapsedAlready := runTests()
	time.Sleep((env.USER_RESET_TIME - elapsedAlready) + (2 * time.Millisecond))
	t.Log("- 2nd batch after the egress has reset -")
	runTests()

	// We want to wait from when the last request was sent, so no need to subtract anything
	time.Sleep(env.GCP_EGRESS_LATENCY)
	assert.Equal(t, int64(0), test.ReadChannel(state.ProvisionalAdditionalEgress))
	assert.Equal(t, totalBodyEgress, state.MeasuredEgress.SimpleRead())
}

func TestDdosSmallFiles(t *testing.T) {
	testDdos(1, false, t)
}
func TestDdosLargerFiles(t *testing.T) {
	testDdos(7_500_000, false, t)
}
func TestDdosReqCountCap(t *testing.T) {
	testDdos(7_500_000, true, t)
}
func testDdos(
	reqSize int, checkingCountCap bool,
	t *testing.T,
) {
	startTime := time.Now().UTC()
	r, state, env := test.InitProgram(&test.Config{
		/*
			Smaller than constants.MIN_REQUEST_EGRESS so the handling of the disparity
			between the cautious total egress and the eventual actual egress can be tested
		*/
		RandomContentLength:     reqSize,
		DisableRequestLog:       true,
		UseLowTotalRequestLimit: checkingCountCap,
	})

	reqSizePlusOverhead := int64(reqSize) + constants.ASSUMED_OVERHEAD
	effectiveSize := max(reqSizePlusOverhead, constants.MIN_REQUEST_EGRESS)
	const CLIENT_COUNT = 100

	requestUntil503 := func() {
		nextIP := make(chan uint)
		go func() { nextIP <- test.FourBytesToUint(1, 1, 1, 1) }()
		makeRequestWithUniqueIP := func() *httptest.ResponseRecorder {
			ip := <-nextIP
			go func() { nextIP <- ip + 1 }()

			req := test.NewRequest("GET", "/v1/object/foo.bar", nil, env)
			req.Header.Set(env.PROXY_ORIGINAL_IP_HEADER_NAME, test.FormatIp(test.UintToFourBytes(ip)))
			return test.FetchUsingRequest(req, r)
		}

		// The requests should be made quick enough here that a lot of them won't be corrected until the loop exits
		maxTime := env.GCP_RESET_TICK_DELAY - (env.GCP_EGRESS_LATENCY + 5)
		var wg sync.WaitGroup
		for i := 0; i < CLIENT_COUNT; i++ {
			wg.Add(1)

			go func() {
				for {
					w := makeRequestWithUniqueIP()
					assert.NotEqual(t, 429, w.Code) // Each user shouldn't send enough to get a 429

					if w.Code == 503 || time.Since(startTime) >= maxTime {
						break
					}
				}
				wg.Done()
			}()
		}
		wg.Wait()
		time.Sleep(2 * time.Millisecond)
		if time.Since(startTime) >= maxTime {
			t.Fatal("couldn't reach total monthly limit before it was reset. Is your computer busy?")
		}

		reqCount := test.ReadChannel(state.MonthlyRequestCount)
		cautiousTotalEgress := state.MeasuredEgress.SimpleRead() + test.ReadChannel(state.ProvisionalAdditionalEgress)
		t.Logf(
			"cautiously high total egress: %v\nrequest count: %v",
			cautiousTotalEgress,
			reqCount,
		)
		if checkingCountCap {
			assert.Equal(t, env.MAX_TOTAL_REQUESTS, reqCount)
		} else {
			assert.Less(t, reqCount, env.MAX_TOTAL_REQUESTS) // The egress should be what capped it
		}

		time.Sleep(env.GCP_EGRESS_LATENCY)
		expectingSpareRequests := false
		if !checkingCountCap {
			cautiousTotalEgress = state.MeasuredEgress.SimpleRead() + test.ReadChannel(state.ProvisionalAdditionalEgress)
			maxOvershoot := env.MAX_TOTAL_EGRESS + (effectiveSize * CLIENT_COUNT) // Because request cancelling isn't emulated
			withinOvershoot := cautiousTotalEgress <= maxOvershoot
			assert.True(t, withinOvershoot)
			t.Logf("cautiously high total egress after env.GCP_EGRESS_LATENCY: %v", cautiousTotalEgress)
			expectingSpareRequests = cautiousTotalEgress+constants.MIN_REQUEST_EGRESS <= env.MAX_TOTAL_EGRESS
		}

		w := makeRequestWithUniqueIP()
		if expectingSpareRequests {
			assert.Equal(t, 200, w.Code)
		} else {
			assert.Equal(t, 503, w.Code)
		}
	}
	requestUntil503()
	// Because of the provisional egress system, egress can be counted twice until env.GCP_EGRESS_LATENCY passes
	// So it should now be possible to make a few more requests
	requestUntil503()

	time.Sleep((env.GCP_RESET_TICK_DELAY - time.Since(startTime)) + env.GCP_EGRESS_LATENCY)
	startTime = startTime.Add(env.GCP_RESET_TICK_DELAY)
	t.Log("- 2nd batch after the month has ended -")
	assert.Equal(t, int64(0), state.MeasuredEgress.SimpleRead())
	assert.Equal(t, int64(0), test.ReadChannel(state.ProvisionalAdditionalEgress))
	assert.Equal(t, int64(0), test.ReadChannel(state.MonthlyRequestCount))
	requestUntil503()
	requestUntil503()
}
