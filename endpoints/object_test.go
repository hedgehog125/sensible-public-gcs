package endpoints_test

import (
	"encoding/json"
	"fmt"
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
func TestContinuousRequestsOfMinSize(t *testing.T) {
	r, _, env := test.InitProgram(&test.Config{
		RandomContentLength: 1, // 1 byte objects
	})

	if env.DAILY_EGRESS_PER_USER%constants.MIN_REQUEST_EGRESS != 0 {
		panic(fmt.Sprintf(
			"env.DAILY_EGRESS_PER_USER (%v) is not divisible by constants.MIN_REQUEST_EGRESS (%v)",
			env.DAILY_EGRESS_PER_USER,
			constants.MIN_REQUEST_EGRESS,
		))
	}

	startTime := time.Now()
	var total int64
	for total = int64(0); total < env.DAILY_EGRESS_PER_USER; {
		w := test.Fetch("GET", "/v1/object/foo.bar", nil, r, env)
		assert.Equal(t, 200, w.Code)
		_ = w.Body.String()

		total += constants.MIN_REQUEST_EGRESS
	}

	w := test.Fetch("GET", "/v1/remaining/egress", nil, r, env)
	assert.Equal(t, 200, w.Code)
	jsonRes := endpoints.RemainingEgressResponse{}
	err := json.NewDecoder(w.Body).Decode(&jsonRes)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), jsonRes.Remaining)
	assert.Equal(t, total, jsonRes.Used)

	if time.Since(startTime) >= env.USER_RESET_TIME-2 {
		panic("couldn't reach daily limit before it was reset. is your computer busy?")
	}

	w = test.Fetch("GET", "/v1/object/foo.bar", nil, r, env)
	assert.Equal(t, 429, w.Code)
	_ = w.Body.String()
}
