package main

import (
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

var env intertypes.Env

func main() {
	env = subfns.LoadEnvironmentVariables()

	subfns.CreateGCSKeyFile()
	bucket := subfns.CreateGCSBucketClient(&env)

	r := subfns.CreateServer()
	subfns.AddMiddleware(r, &env)
	subfns.RegisterEndpoints(r, bucket)
	subfns.StartServer(r, &env)
}
