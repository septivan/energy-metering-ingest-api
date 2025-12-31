package main

import (
	"os"

	"go.uber.org/zap"
)

func newLogger() (*zap.Logger, error) {
	// Use development mode if in local environment
	env := os.Getenv("ENV")
	if env == "development" || env == "dev" || env == "" {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
