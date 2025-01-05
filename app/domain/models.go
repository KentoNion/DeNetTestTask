package domain

import (
	"app/iternal/config"
	"errors"
	"reflect"
	"time"
)

type UserID int64
type UserScore int64
type Email string
type Nickname string

type User struct {
	ID         UserID    `db:"id"`
	Nickname   Nickname  `db:"nickname"`
	Email      Email     `db:"email"`
	Score      UserScore `db:"score"`
	Registered time.Time `db:"registered"`
	InvitedBy  *UserID   `db:"invited_by"`
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
