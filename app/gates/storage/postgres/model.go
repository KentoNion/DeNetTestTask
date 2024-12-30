package storage

import (
	"app/domain"
	sq "github.com/Masterminds/squirrel"
	"github.com/bool64/sqluct"
	"github.com/jmoiron/sqlx"
	"log/slog"
	"time"
)

type Store struct {
	db  *sqlx.DB
	sq  sq.StatementBuilderType
	sm  sqluct.Mapper
	log *slog.Logger
}

type user struct {
	id         domain.UserID    `db:"id"`
	nickname   string           `db:"nickname"`
	email      string           `db:"email"`
	score      domain.UserScore `db:"score"`
	registered time.Time        `db:"registered"`
}
