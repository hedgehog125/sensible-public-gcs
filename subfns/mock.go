package subfns

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

type MockGCPClient struct {
	RandomContent []byte
}

func (client *MockGCPClient) GetEgress(env *intertypes.Env) (int64, error) {
	fmt.Println("TODO")
	return 0, nil
}

// 2nd return value is always true as this never errors
//
// Returns a 200 response with a body of "a"s, totalling 10MB
func (client *MockGCPClient) FetchObject(
	objectPath string,
	ctx *gin.Context,
) (*http.Response, bool) {
	is404Route := objectPath == "404"

	var content []byte
	if is404Route {
		content = []byte("secret")
	} else {
		content = client.RandomContent
	}

	// Adapted from https://stackoverflow.com/questions/33978216/create-http-response-instance-with-sample-body-string-in-golang by boaz_shuster and David G

	length := len(content)
	response := &http.Response{
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(length),
		Header: http.Header{
			"Content-Length": []string{strconv.Itoa(length)},
		},
		Body: io.NopCloser(bytes.NewBuffer(content)),
	}
	if is404Route {
		response.Status = "404 Not Found"
		response.StatusCode = 404
	} else {
		response.Status = "200 OK"
		response.StatusCode = 200
	}

	return response, false
}
func NewMockGCPClient(randomContentLength int) *MockGCPClient {
	client := MockGCPClient{
		RandomContent: bytes.Repeat([]byte("a"), randomContentLength),
	}

	return &client
}
