package storage

import (
	"fmt"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
	storage "github.com/avGenie/go-loyalty-system/internal/app/storage/postgres"
)

func InitStorage(config config.Config) (model.Storage, error) {
	if len(config.DBConnect) == 0 {
		return nil, fmt.Errorf("empty database config")
	}

	return storage.NewPostgresStorage(config.DBConnect)
}
