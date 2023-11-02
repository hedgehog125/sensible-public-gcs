package main

import (
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
	"github.com/hedgeghog125/sensible-public-gcs/subfns"
)

var env intertypes.Env

func main() {
	env = subfns.LoadEnvironmentVariables()
	r := subfns.CreateServer()
	subfns.AddMiddleware(r, &env)
	subfns.RegisterEndpoints(r)
	subfns.StartServer(r, &env)
}
