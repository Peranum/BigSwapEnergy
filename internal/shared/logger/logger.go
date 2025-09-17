package logger

import (
	"go.uber.org/zap"
)

func NewLogger() *zap.Logger {
	config := zap.NewProductionConfig()

	logger, err := config.Build()
	if err != nil {
		panic("Failed to create logger: " + err.Error())
	}

	return logger
}
