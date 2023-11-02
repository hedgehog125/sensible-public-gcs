package intertypes

type Env struct {
	PORT                 int
	CORS_ALLOWED_ORIGINS []string
	GCS_BUCKET_NAME      string

	IS_DEV bool
}
