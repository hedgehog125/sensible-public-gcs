package intertypes

type Env struct {
	PORT                 int
	CORS_ALLOWED_ORIGINS []string
	GCS_BUCKET_NAME      string
	GCP_PROJECT_NAME     string

	DAILY_EGRESS_PER_USER          int64
	MAX_TOTAL_EGRESS               int64
	MEASURE_TOTAL_EGRESS_FROM_ZERO bool
	MAX_TOTAL_REQUESTS             int64

	IS_DEV  bool
	IS_TEST bool
}
