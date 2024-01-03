package endpoints_test

import (
	"fmt"
	"testing"

	"github.com/hedgeghog125/sensible-public-gcs/test"
	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	r, _, _ := test.InitProgram()

	w := test.Fetch("GET", "/health", nil, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "", w.Body.String())
}

const TEST_IP = "42.42.42.42"

func TestIpNoProxy(t *testing.T) {
	r, _, env := test.InitProgram()

	w := test.Fetch("GET", "/v1/ip", nil, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "", w.Body.String())

	req := test.NewRequest("GET", "/v1/ip", nil)
	req.RemoteAddr = fmt.Sprintf("%v:%v", TEST_IP, env.PORT)
	w = test.FetchUsingRequest(req, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, TEST_IP, w.Body.String())
}
func TestIpWithProxy(t *testing.T) {
	r, _, env := test.InitProgram()
	env.PROXY_ORIGINAL_IP_HEADER_NAME = "X-Test-IP"
	r.TrustedPlatform = env.PROXY_ORIGINAL_IP_HEADER_NAME

	req := test.NewRequest("GET", "/v1/ip", nil)
	req.Header.Set(env.PROXY_ORIGINAL_IP_HEADER_NAME, TEST_IP)
	w := test.FetchUsingRequest(req, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, TEST_IP, w.Body.String())
}
