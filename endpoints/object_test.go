package endpoints_test

import (
	"encoding/json"
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
	r, _, env := test.InitProgram(&test.Config{
		RandomContentLength: reqSize,
	})

	effectiveSize := max(int64(reqSize)+constants.ASSUMED_OVERHEAD, constants.MIN_REQUEST_EGRESS)
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

	runTests := func() time.Duration {
		startTime := time.Now().UTC()
		var total int64
		for total = int64(0); total+effectiveSize <= env.DAILY_EGRESS_PER_USER; {
			w := test.Fetch("GET", "/v1/object/foo.bar", nil, r, env)
			assert.Equal(t, 200, w.Code)
			_ = w.Body.String()

			total += effectiveSize
		}

		w := test.Fetch("GET", "/v1/remaining/egress", nil, r, env)
		assert.Equal(t, 200, w.Code)
		jsonRes := endpoints.RemainingEgressResponse{}
		err := json.NewDecoder(w.Body).Decode(&jsonRes)
		assert.Nil(t, err)
		assert.Equal(t, env.DAILY_EGRESS_PER_USER-total, jsonRes.Remaining)
		assert.Equal(t, total, jsonRes.Used)
		// TODO: check total request count is correct?

		if time.Since(startTime) >= env.USER_RESET_TIME-2 {
			t.Fatal("couldn't reach daily limit before it was reset. is your computer busy?")
		}

		expect429 := func() {
			w = test.Fetch("GET", "/v1/object/foo.bar", nil, r, env)
			assert.Equal(t, 429, w.Code)
			_ = w.Body.String()
		}
		expect429()
		time.Sleep(5 * time.Millisecond)
		expect429()

		return time.Since(startTime)
	}

	elapsedAlready := runTests()
	time.Sleep((env.USER_RESET_TIME - elapsedAlready) + (2 * time.Millisecond))
	runTests()
}
