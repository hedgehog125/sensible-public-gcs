package intertypes

import "sync"

type MutexValue[T any] struct {
	Mutex sync.RWMutex
	Value T
}

func (mv *MutexValue[T]) SimpleWrite(value T) {
	mv.Mutex.Lock()
	defer mv.Mutex.Unlock()
	mv.Value = value
}
func (mv *MutexValue[T]) SimpleRead() T {
	mv.Mutex.RLock()
	defer mv.Mutex.RUnlock()
	return mv.Value
}
