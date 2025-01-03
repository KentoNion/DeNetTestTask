package storage

import (
	"app/domain"
	"errors"
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
	nickname   domain.Nickname  `db:"nickname"`
	email      domain.Email     `db:"email"`
	score      domain.UserScore `db:"score"`
	registered time.Time        `db:"registered"`
	invitedBy  domain.UserID    `db:"invited_by"`
}

var ErrUserAlreadyInvited = errors.New("User already invited")
var errNoRowsAffected = errors.New("No rows affected")
