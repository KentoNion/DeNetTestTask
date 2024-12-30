package storage

import (
	"app/domain"
	sq "github.com/Masterminds/squirrel"
	"github.com/bool64/sqluct"
	"github.com/jmoiron/sqlx"
	"log/slog"
)

type Store struct {
	db  *sqlx.DB
	sq  sq.StatementBuilderType
	sm  sqluct.Mapper
	log *slog.Logger
}

func NewDB(db *sqlx.DB, log *slog.Logger) *Store {
	return &Store{
		db:  db,
		sm:  sqluct.Mapper{Dialect: sqluct.DialectPostgres},
		sq:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log: log,
	}
}

// Получение информации по пользователю
func (s *Store) GetUser(id domain.UserID) (domain.User, error) {
	const op = "storage.PostgreSQL.GetUser"

}
