package http

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/auth"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/middleware/logger"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/middleware/token"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/orders"
	storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type HTTPServer struct {
	server *http.Server

	config  config.Config
	storage storage.Storage

	authenticator auth.AuthUser
	orders        orders.Order
}

func New(config config.Config, storage storage.Storage) *HTTPServer {
	authenticator := auth.New(storage)
	order := orders.New(storage, config)

	mux := createMux(authenticator, order)

	server := &http.Server{
		Addr:    config.NetAddr,
		Handler: mux,
	}

	instance := &HTTPServer{
		server:        server,
		config:        config,
		storage:       storage,
		authenticator: authenticator,
		orders:        order,
	}

	return instance
}

func (s *HTTPServer) StartHTTPServer() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	go func() {
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("fatal error while starting server", zap.Error(err))
		}
	}()

	<-ctx.Done()

	zap.L().Info("Got interruption signal. Shutting down HTTP server gracefully...")
	err := s.server.Shutdown(context.Background())
	if err != nil {
		zap.L().Error("error while shutting down server", zap.Error(err))
	}
	s.stop()
}

func (s *HTTPServer) stop() {
	s.orders.Stop()

	zap.L().Info("server has been stopped")
}

func createMux(authenticator auth.AuthUser, orders orders.Order) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.LoggerMiddleware)
	r.Use(token.TokenParserMiddleware)

	r.Post("/api/user/register", authenticator.CreateUser())
	r.Post("/api/user/login", authenticator.AuthenticateUser())
	r.Post("/api/user/orders", orders.UploadOrder())
	r.Post("/api/user/balance/withdraw", orders.WithdrawBonuses())

	r.Get("/api/user/orders", orders.GetUserOrders())
	r.Get("/api/user/withdrawals", orders.GetUserWithdrawals())
	r.Get("/api/user/balance", orders.GetUserBalance())

	return r
}
