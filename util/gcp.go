package util

import (
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func CalculateCautiousEgress(state *intertypes.State) int64 {
	return state.MeasuredEgress.SimpleRead() + ReadAndResendChannel[int64](
		state.ProvisionalAdditionalEgress,
	)
}
