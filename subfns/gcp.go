package subfns

import (
	"context"
	"fmt"
	"os"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/storage"
	"github.com/hedgeghog125/sensible-public-gcs/constants"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/util"
)

func CreateGCSKeyFile() {
	keyValue := os.Getenv("GCS_KEY")
	if keyValue != "" {
		err := os.MkdirAll("secret", constants.NORMAL_PERMISSION)
		if err != nil {
			panic(fmt.Sprintf("couldn't create \"secret\" directory. error:\n%v", err.Error()))
		}

		err = os.WriteFile(constants.GCS_KEY_PATH, []byte(keyValue), constants.NORMAL_PERMISSION)
		if err != nil {
			panic(fmt.Sprintf("unable to write %v. error:\n%v", constants.GCS_KEY_PATH, err.Error()))
		}
	}
}
func CreateGCSBucketClient(env *intertypes.Env) *storage.BucketHandle {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", constants.GCS_KEY_PATH)
	client, err := storage.NewClient(context.Background())
	if err != nil {
		panic(fmt.Sprintf("couldn't create Google Cloud Storage client. error:\n%v", err.Error()))
	}

	bucket := client.Bucket(env.GCS_BUCKET_NAME)
	return bucket
}
func CreateGCPMonitoringClient() *monitoring.QueryClient {
	client, err := monitoring.NewQueryClient(context.Background())
	if err != nil {
		panic(fmt.Sprintf("couldn't create monitoring client. error:\n%v", err.Error()))
	}

	return client
}
func GCPMonitoringTick(client *monitoring.QueryClient, isInitialTick bool, state *intertypes.State, env *intertypes.Env) {
	value, err := util.GetEgress(client, env)
	if isInitialTick {
		fmt.Printf("initial egress: %v\n", value)
		if err != nil {
			panic("initial egress check failed")
		}

		if env.MEASURE_TOTAL_EGRESS_FROM_ZERO { // Otherwise leave it at 0
			state.InitialMeasuredEgress = value
		}
		state.MeasuredEgress = value - state.InitialMeasuredEgress
		return
	}

	if err != nil {
		return
	}
	state.MeasuredEgress = value - state.InitialMeasuredEgress
}
