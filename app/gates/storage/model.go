package storage

import (
	"app/domain"
	"time"
)

type User struct {
	id         domain.UserID    `db:"id"`
	score      domain.UserScore `db:"score"`
	registered time.Time        `db:"registered"`
}
