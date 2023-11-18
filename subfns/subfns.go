package subfns

import (
	"fmt"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/util"
	"github.com/joho/godotenv"
)

func LoadEnvironmentVariables() intertypes.Env {
	_ = godotenv.Load(".env.local.keys")
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load(".env")

	env := intertypes.Env{}

	env.PORT = util.RequireIntEnv("PORT")
	env.CORS_ALLOWED_ORIGINS = util.RequireStrArrEnv("CORS_ALLOWED_ORIGINS")
	env.GCS_BUCKET_NAME = util.RequireEnv("GCS_BUCKET_NAME")
	env.GCP_PROJECT_NAME = util.RequireEnv("GCP_PROJECT_NAME")

	env.DAILY_EGRESS_PER_USER = util.RequireInt64Env("DAILY_EGRESS_PER_USER")
	env.MAX_TOTAL_EGRESS = util.RequireInt64Env("MAX_TOTAL_EGRESS")
	env.MEASURE_TOTAL_EGRESS_FROM_ZERO = util.RequireEnv("MEASURE_TOTAL_EGRESS_FROM_ZERO") == "true"
	env.MAX_TOTAL_REQUESTS = util.RequireInt64Env("MAX_TOTAL_REQUESTS")

	env.IS_DEV = util.RequireEnv("GIN_MODE") == "debug"

	return env
}
func InitState() intertypes.State {
	state := intertypes.State{
		Users:                       make(map[string]*chan *intertypes.User),
		ProvisionalAdditionalEgress: util.Pointer[chan int64](make(chan int64)),
		MonthlyRequestCount:         util.Pointer[chan int64](make(chan int64)),
	}
	go func() { *state.ProvisionalAdditionalEgress <- 0 }()
	go func() { *state.MonthlyRequestCount <- 0 }()

	return state
}
func StartTickFns(mClient *monitoring.QueryClient, state *intertypes.State, env *intertypes.Env) {
	go func() {
		for {
			time.Sleep(time.Minute)
			GCPMonitoringTick(mClient, false, state, env)
		}
	}()
	go func() {
		for {
			time.Sleep(12 * time.Hour)
			UsersTick(state)
		}
	}()
	go func() {
		for {
			now := time.Now()
			startOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			secondsUntilNextMonth := startOfNextMonth.Unix() - now.Unix()

			time.Sleep(time.Duration(secondsUntilNextMonth) * time.Second)
			fmt.Printf("monthly total request count before reset: %v\n", <-*state.MonthlyRequestCount)
			go func() { *state.MonthlyRequestCount <- 0 }()
		}
	}()
}
