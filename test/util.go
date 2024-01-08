package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func Fetch(
	method string, url string, body io.Reader,
	r *gin.Engine, env *intertypes.Env,
) *httptest.ResponseRecorder {
	req := NewRequest(method, url, body, env)
	return FetchUsingRequest(req, r)
}
func FetchUsingRequest(req *http.Request, r *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Use for constant requests. Panics instead of returning an error
func NewRequest(method string, url string, body io.Reader, env *intertypes.Env) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(fmt.Sprintf("couldn't create HTTP request for test. error:\n%v", err.Error()))
	}
	if env.PROXY_ORIGINAL_IP_HEADER_NAME != "" {
		req.Header.Set(env.PROXY_ORIGINAL_IP_HEADER_NAME, TEST_IP)
	}

	return req
}

// This is in the test package as opposed to util as you almost always want to lock the channel for a moment when reading it. But in tests, you usually just want to check a value.
func ReadChannel[T any](c *chan T) T {
	value := <-*c
	defer func() {
		go func() {
			*c <- value
		}()
	}()
	return value
}
