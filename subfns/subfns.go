package subfns

import (
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

	env.IS_DEV = util.RequireEnv("GIN_MODE") == "debug"

	return env
}
