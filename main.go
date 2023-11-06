package main

import (
	"context"
	"fmt"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

var env intertypes.Env

func main() {
	env = subfns.LoadEnvironmentVariables()

	subfns.CreateGCSKeyFile()
	bucket := subfns.CreateGCSBucketClient(&env)

	client, err := monitoring.NewQueryClient(context.Background())
	if err != nil {
		panic(fmt.Sprintf("couldn't create monitoring client. error:\n%v", err.Error()))
	}
	res, err := client.QueryTimeSeries(context.Background(), &monitoringpb.QueryTimeSeriesRequest{
		Name:  fmt.Sprintf("projects/%v", env.GCP_PROJECT_NAME),
		Query: `fetch gcs_bucket::storage.googleapis.com/network/sent_bytes_count | every 10m | within 10m | group_by [], sum(value.sent_bytes_count)`,
		//Query: `fetch gcs_bucket::storage.googleapis.com/network/sent_bytes_count | align delta(10m) | group_by [], sum(val())`,
	}).Next()
	fmt.Println(err)
	if err == nil {
		fmt.Println(res.PointData[0].Values[0].Value, len(res.PointData))
	}

	r := subfns.CreateServer()
	subfns.AddMiddleware(r, &env)
	subfns.RegisterEndpoints(r, bucket)
	subfns.StartServer(r, &env)
}
