package intertypes

import (
	"time"

	"github.com/puzpuzpuz/xsync/v3"
)

type State struct {
	Users *xsync.MapOf[string, chan *User]
	// If env.MEASURE_TOTAL_EGRESS_FROM_ZERO is false, this will be 0
	InitialMeasuredEgress int64
	MeasuredEgress        *MutexValue[int64]
	// MeasuredEgress is usually a few minutes behind the actual egress usage
	//
	// Add this value to MeasuredEgress to get a cautiously large figure for the egress
	ProvisionalAdditionalEgress chan int64
	MonthlyRequestCount         chan int64
}
type User struct {
	EgressUsed int64
	// A Unix epoch
	ResetAt time.Time
}
