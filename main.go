package main

import (
	"time"

	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

var env intertypes.Env

func main() {
	env = subfns.LoadEnvironmentVariables()

	subfns.CreateGCSKeyFile()
	bucket := subfns.CreateGCSBucketClient(&env)
	mClient := subfns.CreateGCPMonitoringClient()
	recordedEgress := int64(0)
	subfns.GCPMonitoringTick(&recordedEgress, mClient, &env)

	r := subfns.CreateServer()
	subfns.AddMiddleware(r, &env)
	subfns.RegisterEndpoints(r, bucket)
	go subfns.StartServer(r, &env)
	for {
		time.Sleep(time.Minute)
		subfns.GCPMonitoringTick(&recordedEgress, mClient, &env)
	}
}
