package intertypes

type State struct {
	Users map[string]*chan *User
	// If env.MEASURE_TOTAL_EGRESS_FROM_ZERO is false, this will be 0
	InitialMeasuredEgress int64
	MeasuredEgress        int64
	// MeasuredEgress is usually a few minutes behind the actual egress usage
	//
	// Add this value to MeasuredEgress to get a cautiously large figure for the egress
	ProvisionalAdditionalEgress *chan int64
}
type User struct {
	EgressUsed int64
	// A Unix epoch
	ResetAt int64
}
