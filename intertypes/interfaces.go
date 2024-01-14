package intertypes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type GCPClient interface {
	FetchObject(objectPath string, ctx *gin.Context) (*http.Response, bool)
	GetEgress(env *Env) (int64, error)
}
