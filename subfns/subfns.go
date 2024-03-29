package subfns

import (
	"fmt"
	"os"
	"time"

	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/util"
	"github.com/joho/godotenv"
	"github.com/puzpuzpuz/xsync/v3"
)

func LoadEnvironmentVariables() *intertypes.Env {
	_ = godotenv.Load(".env.local.keys")
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load(".env")

	env := intertypes.Env{
		GCP_EGRESS_LATENCY:     3 * time.Minute,
		GCP_MONITOR_TICK_DELAY: 1 * time.Minute,
		GCP_RESET_TICK_DELAY:   -1,
		USER_TICK_DELAY:        12 * time.Hour,
		USER_RESET_TIME:        24 * time.Hour,
	}

	env.PORT = util.RequireIntEnv("PORT")
	env.CORS_ALLOWED_ORIGINS = util.RequireStrArrEnv("CORS_ALLOWED_ORIGINS")
	env.PROXY_ORIGINAL_IP_HEADER_NAME = util.RequireEnv("PROXY_ORIGINAL_IP_HEADER_NAME")

	env.GCS_BUCKET_NAME = util.RequireEnv("GCS_BUCKET_NAME")
	env.GCP_PROJECT_NAME = util.RequireEnv("GCP_PROJECT_NAME")

	env.DAILY_EGRESS_PER_USER = util.RequireInt64Env("DAILY_EGRESS_PER_USER")
	env.MAX_TOTAL_EGRESS = util.RequireInt64Env("MAX_TOTAL_EGRESS")
	env.MEASURE_TOTAL_EGRESS_FROM_ZERO = util.RequireEnv("MEASURE_TOTAL_EGRESS_FROM_ZERO") == "true"
	env.MAX_TOTAL_REQUESTS = util.RequireInt64Env("MAX_TOTAL_REQUESTS")

	env.IS_PROXY_TEST = os.Getenv("IS_PROXY_TEST") == "true"
	env.IS_DEV = util.RequireEnv("GIN_MODE") == "debug"
	env.IS_TEST = os.Getenv("IS_TEST") == "true"

	return &env
}
func InitState() *intertypes.State {
	state := intertypes.State{
		Users:                       xsync.NewMapOf[string, chan *intertypes.User](),
		ProvisionalAdditionalEgress: make(chan int64),
		MonthlyRequestCount:         make(chan int64),
		MeasuredEgress:              &intertypes.MutexValue[int64]{},
	}
	go func() { state.ProvisionalAdditionalEgress <- 0 }()
	go func() { state.MonthlyRequestCount <- 0 }()

	return &state
}
func StartTickFns(client intertypes.GCPClient, state *intertypes.State, env *intertypes.Env) {
	go func() {
		elapsed := time.Duration(0)
		for {
			time.Sleep(env.GCP_MONITOR_TICK_DELAY - elapsed)
			startTime := time.Now().UTC()
			GCPMonitoringTick(client, false, state, env)
			elapsed = time.Since(startTime)
		}
	}()
	go func() {
		elapsed := time.Duration(0)
		for {
			time.Sleep(env.USER_TICK_DELAY - elapsed)
			startTime := time.Now().UTC()
			UsersTick(state, env)
			elapsed = time.Since(startTime)
		}
	}()
	go func() {
		for {
			if env.GCP_RESET_TICK_DELAY == -1 {
				now := time.Now().UTC()
				startOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
				timeUntilNextMonth := startOfNextMonth.Sub(now)

				time.Sleep(timeUntilNextMonth)
			} else {
				time.Sleep(env.GCP_RESET_TICK_DELAY)
			}

			go func() {
				// I think consuming the channel first is necessary?
				fmt.Printf("monthly total request count before reset: %v\n", <-state.MonthlyRequestCount)
				state.MonthlyRequestCount <- 0
			}()
		}
	}()
}
