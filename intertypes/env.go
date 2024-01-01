package intertypes

import "time"

type Env struct {
	PORT                          int
	CORS_ALLOWED_ORIGINS          []string
	PROXY_ORIGINAL_IP_HEADER_NAME string

	GCS_BUCKET_NAME  string
	GCP_PROJECT_NAME string

	DAILY_EGRESS_PER_USER          int64
	MAX_TOTAL_EGRESS               int64
	MEASURE_TOTAL_EGRESS_FROM_ZERO bool
	MAX_TOTAL_REQUESTS             int64

	IS_PROXY_TEST bool
	IS_DEV        bool
	IS_TEST       bool

	// Not an actual environment variable, just uses different constants when testing
	GCP_EGRESS_LATENCY time.Duration
	// Not an actual environment variable, just uses different constants when testing
	GCP_MONITOR_TICK_DELAY time.Duration
	// Not an actual environment variable, just uses different constants when testing
	//
	// GCP resets are when the total egress is reset. -1 means at the start of the month
	GCP_RESET_TICK_DELAY time.Duration
	// Not an actual environment variable, just uses different constants when testing
	USER_TICK_DELAY time.Duration
	// Not an actual environment variable, just uses different constants when testing
	USER_RESET_TIME time.Duration
}
