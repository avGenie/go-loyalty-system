package logger

import (
	"fmt"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"go.uber.org/zap"
)

func Initialize(config config.Config) error {
	level, err := zap.ParseAtomicLevel(config.LogLevel)
	if err != nil {
		return fmt.Errorf("error while setting atomic level to zap logger")
	}

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = level

	log, err := zapConfig.Build()
	if err != nil {
		return fmt.Errorf("error while building zap logger")
	}

	zap.ReplaceGlobals(log)

	return nil
}
