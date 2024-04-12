package http

import (
	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/middleware/auth"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/middleware/logger"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/post"
	storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
	"github.com/go-chi/chi/v5"
)

func CreateRouter(config config.Config, storage storage.Storage) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.LoggerMiddleware)
	r.Use(auth.AuthMiddleware)

	r.Post("/api/user/register", post.CreateUser(storage))

	return r
}
