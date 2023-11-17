package main

import (
	"time"

	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

var env intertypes.Env

func main() {
	env = subfns.LoadEnvironmentVariables()
	state := subfns.InitState()

	subfns.CreateGCSKeyFile()
	bucket := subfns.CreateGCSBucketClient(&env)
	mClient := subfns.CreateGCPMonitoringClient()
	subfns.GCPMonitoringTick(mClient, true, &state, &env)

	r := subfns.CreateServer()
	subfns.AddMiddleware(r, &env)
	subfns.RegisterEndpoints(r, bucket, &state, &env)
	go subfns.StartServer(r, &env)
	for {
		time.Sleep(time.Minute)
		subfns.GCPMonitoringTick(mClient, false, &state, &env)
	}
}
