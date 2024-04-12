package main

import (
	"net/http"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/logger"
	router "github.com/avGenie/go-loyalty-system/internal/app/controller/http/router"
	storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api"
	"go.uber.org/zap"
)

func main() {
	config := config.InitConfig()

	err := logger.Initialize(config)
	if err != nil {
		panic(err)
	}

	storage, err := storage.InitStorage(config)
	if err != nil {
		zap.L().Fatal("failed to init storage", zap.Error(err))
	}
	defer storage.Close()

	err = http.ListenAndServe(config.NetAddr, router.CreateRouter(config, storage))
	if err != nil && err != http.ErrServerClosed {
		zap.L().Fatal("error while starting server", zap.Error(err))
	}
}
