package subfns

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"cloud.google.com/go/storage"
	"github.com/hedgeghog125/sensible-public-gcs/constants"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"google.golang.org/api/iterator"
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

type GCPClient struct {
	bucket  *storage.BucketHandle
	mClient *monitoring.QueryClient
}

// For this billing cycle, doesn't subtract the initial value
func (client *GCPClient) GetEgress(env *intertypes.Env) (int64, error) {
	now := time.Now()
	startOfTheMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	minutesSinceBillingStart := (now.Unix() - startOfTheMonth.Unix()) / 60

	query := fmt.Sprintf(
		`fetch gcs_bucket::storage.googleapis.com/network/sent_bytes_count | every %vm | within %vm | group_by [], sum(value.sent_bytes_count)`,
		minutesSinceBillingStart,
		minutesSinceBillingStart,
	)
	res, err := client.mClient.QueryTimeSeries(context.Background(), &monitoringpb.QueryTimeSeriesRequest{
		Name:  fmt.Sprintf("projects/%v", env.GCP_PROJECT_NAME),
		Query: query,
	}).Next()

	if err != nil {
		if errors.Is(err, iterator.Done) {
			return 0, nil
		}
		return 0, err
	}

	if len(res.PointData) != 1 || len(res.PointData[0].Values) != 1 {
		return 0, errors.New("couldn't get egress because the PointData was an unexpected shape")
	}
	value, ok := res.PointData[0].Values[0].Value.(*monitoringpb.TypedValue_Int64Value)
	if !ok {
		return 0, errors.New("couldn't get egress because the single data point was the wrong type")
	}

	return value.Int64Value, nil
}
func (client *GCPClient) SignedURL(object string, opts *storage.SignedURLOptions) (string, error) {
	return client.bucket.SignedURL(object, opts)
}
func CreateGCPClient(bucket *storage.BucketHandle, mClient *monitoring.QueryClient) intertypes.GCPClient {
	client := GCPClient{
		bucket:  bucket,
		mClient: mClient,
	}

	return &client
}
func GCPMonitoringTick(client intertypes.GCPClient, isInitialTick bool, state *intertypes.State, env *intertypes.Env) {
	value, err := client.GetEgress(env)
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
