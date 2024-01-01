package main_test

import (
	"os"
	"testing"

	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

func TestMain(m *testing.M) {
	os.Setenv("IS_TEST", "true")
	os.Setenv("GIN_MODE", "release")
	env := intertypes.Env{
		PORT:                          8000,
		CORS_ALLOWED_ORIGINS:          []string{"*"},
		PROXY_ORIGINAL_IP_HEADER_NAME: "",

		DAILY_EGRESS_PER_USER:          500000000,
		MAX_TOTAL_EGRESS:               15000000000,
		MEASURE_TOTAL_EGRESS_FROM_ZERO: true,
		MAX_TOTAL_REQUESTS:             50000,

		IS_PROXY_TEST: false,
		IS_TEST:       true,
		IS_DEV:        false,
	}
	state := subfns.InitState()
	client := subfns.CreateMockGCPClient()

	r := subfns.CreateServer(&env)
	subfns.AddMiddleware(r, &env)
	subfns.RegisterEndpoints(r, client, state, &env)
	subfns.StartTickFns(client, state, &env)
	go subfns.StartServer(r, &env)
}
