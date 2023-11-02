package main

import (
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

var env intertypes.Env

func main() {
	env = subfns.LoadEnvironmentVariables()

	subfns.CreateGCSKeyFile()
	bucket := subfns.CreateGCSBucketClient(&env)

	fmt.Println(bucket.SignedURL("...", &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(10 * time.Second),
	}))

	r := subfns.CreateServer()
	subfns.AddMiddleware(r, &env)
	subfns.RegisterEndpoints(r)
	subfns.StartServer(r, &env)
}
