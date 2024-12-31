package storage

import (
	"app/domain"
	"context"
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/bool64/sqluct"
	"github.com/jmoiron/sqlx"
	"log/slog"
)

func NewDB(db *sqlx.DB, log *slog.Logger) *Store {
	return &Store{
		db:  db,
		sm:  sqluct.Mapper{Dialect: sqluct.DialectPostgres},
		sq:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log: log,
	}
}

// добавление нового пользователя
func (p *Store) AddUser(ctx context.Context, user domain.User) error {
	const op = "storage.Postgres.AddUser"
	p.log.Debug(fmt.Sprintf("%v: trying to add new user", op))
	query := p.sm.Insert(p.sq.Insert("users"), user, sqluct.InsertIgnore)
	qry, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, err := p.db.ExecContext(ctx, qry, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rows, _ := rows.RowsAffected(); rows == 0 {
		return fmt.Errorf("%s: %w", op, sql.ErrNoRows)
	}
	p.log.Debug(fmt.Sprintf("%v: sucessfully added new user", op))
	return nil
}

// Получение информации по пользователю
func (p *Store) GetUser(ctx context.Context, id domain.UserID) (domain.User, error) {
	const op = "storage.PostgreSQL.GetUser"
	p.log.Debug(fmt.Sprintf("%v: trying to get info for user %v", op, id))
	query := p.sm.Select(p.sq.Select(), &user{}).From("users").Where(sq.Eq{"id": id})
	qry, args, err := query.ToSql()
	var user domain.User
	if err != nil {
		return user, fmt.Errorf("%s: %v", op, err)
	}
	err = p.db.SelectContext(ctx, &user, qry, args...)
	if err != nil {
		return user, fmt.Errorf("%s: %v", op, err)
	}
	p.log.Debug(fmt.Sprintf("%v: successfully retrieved info for user %v", op, id))
	return user, nil
}

// Получение пользователей
// todo реализовать сортировку по имени, кол-во очков, дате регистрации, прикрутить опциональную пагинацию
func (p *Store) GetUsers(ctx context.Context) ([]user, error) {
	const op = "storage.PostgreSQL.GetUsers"
	p.log.Debug(fmt.Sprintf("%v: trying to get all users", op))
	query := p.sm.Select(p.sq.Select(), &user{}).From("users")
	qry, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %v", op, err)
	}
	var users []user
	err = p.db.SelectContext(ctx, &users, qry, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", op, err)
	}
	p.log.Debug(fmt.Sprintf("%v: success, all users retrieved", op))
	return users, nil
}

// добавление score для user по id
func (p *Store) AddScore(ctx context.Context, id domain.UserID, points int) error {
	const op = "storage.PostgreSQL.AddScore"
	p.log.Debug(fmt.Sprintf("%v: trying to add points (%v) to user (%v) score", op, points, id))
	query := p.sq.Update("users").
		Set("score", sq.Expr("score + ?", points)).
		Where(sq.Eq{"id": id})
	qry, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	res, err := p.db.ExecContext(ctx, qry, args...)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: no rows affected, user not found", op)
	}
	p.log.Debug(fmt.Sprintf("%v: successfully added points (%v) to user (%v)", op, points, id))
	return nil
}

func (p *Store) SetInvitedBy(ctx context.Context, userID, invitedByID domain.UserID) error {
	const op = "storage.PostgreSQL.SetInvitedBy"
	p.log.Debug(fmt.Sprintf("%v: trying to set invited_by for user %v to %v", op, userID, invitedByID))
	query := p.sq.Update("users").
		Set("invited_by", invitedByID).
		Where(sq.And{
			sq.Eq{"id": userID},
			sq.Expr("invited_by IS NULL"), // Условие установки: если строка приглашения пустая, тогда можно писать
		})
	qry, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build query: %v", op, err)
	}
	res, err := p.db.ExecContext(ctx, qry, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute query: %v", op, err)
	}
	// Проверка на то что строка была обновлена:
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get rows affected: %v", op, err)
	}
	if rowsAffected == 0 {
		return ErrUserAlreadyInvited //Если cтрока не была изменена, значит поле invited_by уже было заполнено
	}
	p.log.Debug(fmt.Sprintf("%v: successfully set invited_by for user %v to %v", op, userID, invitedByID))
	return nil
}
