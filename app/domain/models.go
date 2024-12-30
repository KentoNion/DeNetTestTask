package domain

import (
	"app/iternal/config"
	"context"
	"log/slog"
	"time"
)

type UserID string
type UserScore int

type UserService struct {
	store UserStore
	log   *slog.Logger
	cfg   *config.Config
}

type UserStore interface {
	GetUser(ctx context.Context, userID UserID) (User, error)
	NewUser()
	GetUsers()
	AddPoints()
}

func NewUserService(store UserStore, log *slog.Logger, cfg *config.Config) *UserService {
	return &UserService{
		store: store,
		log:   log,
		cfg:   cfg,
	}
}

type User struct {
	id         UserID
	score      UserScore
	registered time.Time
}
