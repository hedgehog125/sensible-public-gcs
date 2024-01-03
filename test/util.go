package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func Fetch(method string, url string, body io.Reader, r *gin.Engine) *httptest.ResponseRecorder {
	req := NewRequest(method, url, body)
	return FetchUsingRequest(req, r)
}
func FetchUsingRequest(req *http.Request, r *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Use for constant requests. Panics instead of returning an error
func NewRequest(method string, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(fmt.Sprintf("couldn't create HTTP request for test. error:\n%v", err.Error()))
	}

	return req
}
