package main

import (
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

func main() {
	env := subfns.LoadEnvironmentVariables()
	state := subfns.InitState()

	subfns.CreateGCSKeyFile()
	bucket := subfns.CreateGCSBucketClient(env)
	mClient := subfns.CreateGCPMonitoringClient()
	client := subfns.CreateGCPClient(bucket, mClient)
	subfns.GCPMonitoringTick(client, true, state, env)

	r := subfns.CreateServer(env)
	subfns.AddMiddleware(r, env)
	subfns.RegisterEndpoints(r, client, state, env)
	subfns.StartTickFns(client, state, env)
	subfns.StartServer(r, env)
}
