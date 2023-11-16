package intertypes

import "sync"

type MutexValue[T any] struct {
	Mutex sync.Mutex
	Value T
}
