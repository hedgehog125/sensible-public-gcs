package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("IS_TEST", "true")
	// TODO
}
