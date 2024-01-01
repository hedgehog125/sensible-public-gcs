package intertypes

import (
	"cloud.google.com/go/storage"
)

type GCPClient interface {
	SignedURL(object string, opts *storage.SignedURLOptions) (string, error)
	GetEgress(env *Env) (int64, error)
}
