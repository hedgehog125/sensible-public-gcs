package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func FetchRoute(method string, url string, body io.Reader, r *gin.Engine) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(fmt.Sprintf("couldn't create HTTP request for test. error:\n%v", err.Error()))
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
