package domain

import (
	"app/iternal/config"
	"errors"
	"reflect"
	"time"
)

type UserID string
type UserScore int
type Email string
type Nickname string
type Filter string

type User struct {
	Id         UserID
	Nickname   Nickname
	Email      Email
	Score      UserScore
	Registered time.Time
	InvitedBy  UserID
}

var ErrNotEmail = errors.New("Wrong format of email")
var ErrNotExistingReward = errors.New("This reward does not exist")
var ErrNoRewardRef = errors.New("No reward for inviting found")

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
