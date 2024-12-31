package domain

import (
	"app/iternal/config"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"time"
)

type UserID string
type UserScore int
type Email string
type Nickname string

type UserService struct {
	store   UserStore
	log     *slog.Logger
	cfg     *config.Config
	rewards map[string]int
}

type UserStore interface {
	GetUser(ctx context.Context, userID UserID) (User, error)
	NewUser()
	GetUsers()
	AddPoints()
}

func NewUserService(store UserStore, log *slog.Logger, cfg *config.Config) *UserService {
	return &UserService{
		store:   store,
		log:     log,
		cfg:     cfg,
		rewards: initRewards(cfg),
	}
}

type User struct {
	Id         UserID
	Nickname   Nickname
	Email      Email
	Score      UserScore
	Registered time.Time
	InvitedBy  UserID
}

var ErrNotEmail = errors.New("not email")

// функция инициализирующая мапу наград из конфига. Размер награды можно изменять config.yaml
func initRewards(cfg *config.Config) map[string]int {
	Rewards := make(map[string]int)

	// Используем рефлексию для извлечения названий полей и их значений
	v := reflect.ValueOf(cfg.Rewards)
	t := reflect.TypeOf(cfg.Rewards)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i).Name              // Имя поля
		value := v.Field(i).Interface().(int) // Значение поля
		Rewards[field] = value
	}
	return Rewards
}
