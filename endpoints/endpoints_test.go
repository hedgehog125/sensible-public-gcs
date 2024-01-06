package endpoints_test

import (
	"fmt"
	"testing"

	"github.com/hedgeghog125/sensible-public-gcs/test"
	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	r, _, env := test.InitProgram(nil)

	w := test.Fetch("GET", "/health", nil, r, env)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "", w.Body.String())
}

func TestIpNoProxy(t *testing.T) {
	r, _, env := test.InitProgram(&test.Config{
		DisableProxy: true,
	})

	w := test.Fetch("GET", "/v1/ip", nil, r, env)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "", w.Body.String())

	req := test.NewRequest("GET", "/v1/ip", nil, env)
	req.RemoteAddr = fmt.Sprintf("%v:%v", test.TEST_IP, env.PORT)
	w = test.FetchUsingRequest(req, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, test.TEST_IP, w.Body.String())
}
func TestIpWithProxy(t *testing.T) {
	r, _, env := test.InitProgram(nil)

	w := test.Fetch("GET", "/v1/ip", nil, r, env)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, test.TEST_IP, w.Body.String())
}
