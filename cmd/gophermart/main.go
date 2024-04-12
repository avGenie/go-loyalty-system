package main

import (
	"net/http"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/logger"
	"go.uber.org/zap"
)

func main() {
	config := config.InitConfig()

	err := logger.Initialize(config)
	if err != nil {
		panic(err)
	}

	// init storage

	// init router

	err = http.ListenAndServe(config.NetAddr, nil)
	if err != nil && err != http.ErrServerClosed {
		zap.L().Fatal("error while starting server", zap.Error(err))
	}
}
