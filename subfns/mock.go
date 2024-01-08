package subfns

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/constants"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/util"
)

type MockGCPClient struct {
	RandomContent  []byte
	MeasuredEgress *chan int64
	Env            *intertypes.Env
}

func (client *MockGCPClient) GetEgress(env *intertypes.Env) (int64, error) {
	value := <-*client.MeasuredEgress
	go func() { *client.MeasuredEgress <- value }()

	return value, nil
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

	go func() {
		time.Sleep(
			time.Duration(float64(client.Env.GCP_MONITOR_TICK_DELAY.Nanoseconds()) * (rand.Float64() + 1)),
		)

		measuredEgress := <-*client.MeasuredEgress
		measuredEgress += int64(length) + constants.ASSUMED_OVERHEAD
		*client.MeasuredEgress <- measuredEgress
	}()

	return response, false
}
func NewMockGCPClient(randomContentLength int, env *intertypes.Env) *MockGCPClient {
	client := MockGCPClient{
		RandomContent:  bytes.Repeat([]byte("a"), randomContentLength),
		MeasuredEgress: util.Pointer(make(chan int64)),
		Env:            env,
	}
	go func() { *client.MeasuredEgress <- 0 }()

	go func() {
		for {
			time.Sleep(env.GCP_RESET_TICK_DELAY)
			go func() {
				<-*client.MeasuredEgress
				*client.MeasuredEgress <- 0
			}()
		}
	}()

	return &client
}
