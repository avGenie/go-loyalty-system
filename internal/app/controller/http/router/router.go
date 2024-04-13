package http

import (
	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/middleware/token"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/middleware/logger"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/auth"
	storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
	"github.com/go-chi/chi/v5"
)

func CreateRouter(config config.Config, storage storage.Storage) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.LoggerMiddleware)
	r.Use(token.TokenParserMiddleware)

	r.Post("/api/user/register", auth.CreateUser(storage))

	return r
}
