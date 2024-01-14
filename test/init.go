package test

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hedgeghog125/sensible-public-gcs/constants"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

type Config struct {
	RandomContentLength     int
	DisableProxy            bool
	DisableRequestLog       bool
	UseLowTotalRequestLimit bool
}

func InitProgram(config *Config) (*gin.Engine, *intertypes.State, *intertypes.Env) {
	if config == nil {
		config = &Config{}
	}
	if config.RandomContentLength == -1 {
		config.RandomContentLength = DEFAULT_RANDOM_CONTENT_LENGTH
	}

	os.Setenv("IS_TEST", "true")
	os.Setenv("GIN_MODE", "release")
	gin.SetMode(gin.ReleaseMode)

	maxTotalRequests := int64(500_000)
	maxTotalEgress := 1000 * max(int64(config.RandomContentLength), constants.MIN_REQUEST_EGRESS)
	if config.UseLowTotalRequestLimit {
		maxTotalRequests = 1000
		maxTotalEgress *= 100 // So the total egress isn't the limiting factor
	}
	env := intertypes.Env{
		PORT:                          8000,
		CORS_ALLOWED_ORIGINS:          []string{"*"},
		PROXY_ORIGINAL_IP_HEADER_NAME: "",

		DAILY_EGRESS_PER_USER:          15_000_000, // 15MB
		MAX_TOTAL_EGRESS:               maxTotalEgress,
		MEASURE_TOTAL_EGRESS_FROM_ZERO: true,
		MAX_TOTAL_REQUESTS:             maxTotalRequests,

		IS_PROXY_TEST: false,
		IS_TEST:       true,
		IS_DEV:        false,

		// Overwritten constants
		DISABLE_REQUEST_LOGS:   config.DisableRequestLog,
		GCP_EGRESS_LATENCY:     30 * time.Millisecond,
		GCP_MONITOR_TICK_DELAY: 10 * time.Millisecond,
		GCP_RESET_TICK_DELAY:   3 * time.Second,        // Instead of at the start of the month
		USER_TICK_DELAY:        750 * time.Millisecond, // 250ms would keep the original proportion with USER_RESET_TIME, but having the user around longer is useful
		USER_RESET_TIME:        500 * time.Millisecond,
	}
	if !config.DisableProxy {
		env.PROXY_ORIGINAL_IP_HEADER_NAME = PROXY_ORIGINAL_IP_HEADER_NAME
	}
	state := subfns.InitState()
	client := subfns.NewMockGCPClient(config.RandomContentLength, &env)

	r := subfns.CreateServer(&env)
	subfns.AddMiddleware(r, &env)
	subfns.RegisterEndpoints(r, client, state, &env)
	subfns.StartTickFns(client, state, &env)
	go subfns.StartServer(r, &env)

	return r, state, &env
}
