package logger

import (
	"os"

	"go.uber.org/zap"
)

func NewLogger() *zap.Logger {
	config := zap.NewProductionConfig()

	if os.Getenv("DEBUG") == "true" {
		config = zap.NewDevelopmentConfig()
	}

	logger, err := config.Build()
	if err != nil {
		panic("Failed to create logger: " + err.Error())
	}

	return logger
}
