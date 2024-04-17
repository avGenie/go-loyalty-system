package http

import (
	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/auth"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/middleware/logger"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/middleware/token"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/orders"
	storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
	"github.com/go-chi/chi/v5"
)

func CreateRouter(config config.Config, storage storage.Storage) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.LoggerMiddleware)
	r.Use(token.TokenParserMiddleware)

	authenticator := auth.New(storage)
	orders := orders.New(storage)

	r.Post("/api/user/register", authenticator.CreateUser())
	r.Post("/api/user/login", authenticator.AuthenticateUser())
	r.Post("/api/user/orders", orders.UploadOrder())

	r.Get("/api/user/orders", orders.GetUserOrders())

	return r
}
