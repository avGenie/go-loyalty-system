package main

import (
	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/logger"
	storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api"
	http_server "github.com/avGenie/go-loyalty-system/internal/app/controller/http/server"
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

	zap.L().Info("start gophermart server")

	server := http_server.New(config, storage)
	server.StartHTTPServer()
}
