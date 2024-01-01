package subfns

import (
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

type MockGCPClient struct {
}

func (client *MockGCPClient) GetEgress(env *intertypes.Env) (int64, error) {
	fmt.Println("TODO")
	return 0, nil
}
func (client *MockGCPClient) SignedURL(object string, opts *storage.SignedURLOptions) (string, error) {
	fmt.Println("TODO")
	return "", nil
}
func CreateMockGCPClient() *MockGCPClient {
	client := MockGCPClient{}

	return &client
}
