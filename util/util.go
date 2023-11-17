package util

import "math/rand"

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	randomString := make([]byte, length)

	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}

	return string(randomString)
}
func ReadAndResendChannel[T any](channel *chan T) T {
	value := <-*channel
	go func() { *channel <- value }()
	return value
}
func Pointer[T any](value T) *T {
	return &value
}
