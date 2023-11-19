package util

import (
	"context"
	"errors"
	"fmt"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"google.golang.org/api/iterator"
)

// For this billing cycle, doesn't subtract the initial value
func GetEgress(client *monitoring.QueryClient, env *intertypes.Env) (int64, error) {
	now := time.Now()
	startOfTheMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	minutesSinceBillingStart := (now.Unix() - startOfTheMonth.Unix()) / 60

	query := fmt.Sprintf(
		`fetch gcs_bucket::storage.googleapis.com/network/sent_bytes_count | every %vm | within %vm | group_by [], sum(value.sent_bytes_count)`,
		minutesSinceBillingStart,
		minutesSinceBillingStart,
	)
	res, err := client.QueryTimeSeries(context.Background(), &monitoringpb.QueryTimeSeriesRequest{
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
func CalculateCautiousEgress(state *intertypes.State) int64 {
	return state.MeasuredEgress + ReadAndResendChannel[int64](
		state.ProvisionalAdditionalEgress,
	)
}
