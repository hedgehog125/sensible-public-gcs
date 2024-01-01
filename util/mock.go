package util

import (
	"time"

	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

var testSleepRemappings = map[time.Duration]time.Duration{
	1 * time.Minute: 5 * time.Millisecond,
	3 * time.Minute: 15 * time.Millisecond,
	12 * time.Hour:  1 * time.Second,
}

// Sleeps for the duration provided
//
// During tests, the duration will be remapped to something shorter
func Sleep(d time.Duration, env *intertypes.Env) {
	if env.IS_TEST {
		time.Sleep(testSleepRemappings[d])
		return
	}

	time.Sleep(d)
}

// Always sleeps for the duration, even while testing
func SleepConst(d time.Duration) {
	time.Sleep(d)
}
