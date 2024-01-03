package endpoints_test

import (
	"testing"

	"github.com/hedgeghog125/sensible-public-gcs/test"
	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	r, _ := test.InitProgram()

	w := test.FetchRoute("GET", "/health", nil, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "", w.Body.String())
}
