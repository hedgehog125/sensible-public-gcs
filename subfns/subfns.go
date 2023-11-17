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
	env.GCP_PROJECT_NAME = util.RequireEnv("GCP_PROJECT_NAME")

	env.DAILY_EGRESS_PER_USER = int64(util.RequireIntEnv("DAILY_EGRESS_PER_USER"))
	env.MAX_TOTAL_EGRESS = int64(util.RequireIntEnv("MAX_TOTAL_EGRESS"))
	env.MEASURE_TOTAL_EGRESS_FROM_ZERO = util.RequireEnv("MEASURE_TOTAL_EGRESS_FROM_ZERO") == "true"

	env.IS_DEV = util.RequireEnv("GIN_MODE") == "debug"

	return env
}
func InitState() intertypes.State {
	state := intertypes.State{
		Users:                       make(map[string]*chan *intertypes.User),
		ProvisionalAdditionalEgress: util.Pointer[chan int64](make(chan int64)),
	}
	go func() { *state.ProvisionalAdditionalEgress <- 0 }()

	return state
}
