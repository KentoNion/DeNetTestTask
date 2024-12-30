package server

import (
	"app/domain"
	"app/iternal/config"
	"context"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
)

type Server struct {
	db      domain.UserStore
	context context.Context
	log     *slog.Logger
	srv     *domain.UserService
}

func NewServer(db domain.UserStore, cfg *config.Config, log *slog.Logger, router *chi.Mux) *Server {
	server := &Server{ //формируем структуру сервера
		db:      db,
		context: context.Background(),
		log:     log,
		srv:     domain.NewUserService(db, log, cfg),
	}
	//роутим эндпоинты
	router.Method(http.MethodPost, "/login", http.HandlerFunc(server.loginHandler))
	router.Method(http.MethodPost, "/refresh", http.HandlerFunc(server.refreshHandler))
	server.log.Info("router configured")
	return server
}
