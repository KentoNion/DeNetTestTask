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
		p.log.Error(op, err)
		return err
	}
	rows, err := p.db.ExecContext(ctx, qry, args...)
	if err != nil {
		p.log.Error(op, err)
		return err
	}
	if rows, _ := rows.RowsAffected(); rows == 0 {
		p.log.Error(op, "no rows affected")
		return errNoRowsAffected
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
	if err == sql.ErrNoRows {
		p.log.Debug("user not found")
		return user, sql.ErrNoRows
	}
	if err != nil {
		p.log.Error(op, err)
		return user, err
	}
	err = p.db.SelectContext(ctx, &user, qry, args...)
	if err != nil {
		p.log.Error(op, err)
		return user, err
	}
	p.log.Debug(fmt.Sprintf("%v: successfully retrieved info for user %v", op, id))
	return user, nil
}

// Получение пользователей
func (p *Store) GetUsers(ctx context.Context, filter domain.Sorter, page int, limit int) ([]domain.User, error) {
	const op = "storage.PostgreSQL.GetUsers"
	var users []domain.User
	p.log.Debug(fmt.Sprintf("%v: trying to get all users", op))
	query := p.sm.Select(p.sq.Select(), &user{}).From("users")

	//фильтрация 0-Рейтинг, 1-алфавит(никнейм), 2-id/дате регистрации
	switch filter {
	case "score":
		query = query.OrderBy("score DESC")
	case "nickname":
		query = query.OrderBy("nickname ASK")
	case "id":
		query = query.OrderBy("id ASK")
	default:
		query = query.OrderBy("id ASK")
	}

	//Опциональная пагинация
	if limit != 0 {
		offset := (page - 1) * limit
		query = query.Offset(uint64(offset)).Limit(uint64(limit))
	}

	qry, args, err := query.ToSql()
	if err != nil {
		p.log.Error(op, err)
		return nil, err
	}
	err = p.db.SelectContext(ctx, &users, qry, args...)
	if err != nil {
		p.log.Error(op, err)
		return nil, err
	}
	p.log.Debug(fmt.Sprintf("%v: success, all users retrieved", op))
	return users, nil
}

// добавление score для user по id
func (p *Store) AddPoints(ctx context.Context, id domain.UserID, points int) error {
	const op = "storage.PostgreSQL.AddScore"
	p.log.Debug(fmt.Sprintf("%v: trying to add points (%v) to user (%v) score", op, points, id))
	query := p.sq.Update("users").
		Set("score", sq.Expr("score + ?", points)).
		Where(sq.Eq{"id": id})
	qry, args, err := query.ToSql()
	if err != nil {
		p.log.Error(op, err)
		return err
	}
	res, err := p.db.ExecContext(ctx, qry, args...)
	if err != nil {
		p.log.Error(op, err)
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		p.log.Error(op, err)
		return err
	}
	if rowsAffected == 0 {
		p.log.Error(op, "no rows affected")
		return errNoRowsAffected
	}
	p.log.Debug(fmt.Sprintf("%v: successfully added points (%v) to user (%v)", op, points, id))
	return nil
}

func (p *Store) SetInvitedBy(ctx context.Context, userID, invitedByID domain.UserID) error {
	const op = "storage.PostgreSQL.SetInvitedBy"
	p.log.Debug(fmt.Sprintf("%v: trying to set invited_by for user %v to %v", op, userID, invitedByID))
	_, err := p.GetUser(ctx, invitedByID) //todo: костыль для проверки существования пользователя, вообще эту функцию нужно пересобрать в транзакцию
	if err != nil {
		p.log.Error(op, err)
		return err
	}
	query := p.sq.Update("users").
		Set("invited_by", invitedByID).
		Where(sq.And{
			sq.Eq{"id": userID},
			sq.Expr("invited_by IS NULL"), // Условие установки: если строка приглашения пустая, тогда можно писать
		})
	qry, args, err := query.ToSql()
	if err != nil {
		p.log.Error(op, err)
		return err
	}
	res, err := p.db.ExecContext(ctx, qry, args...)
	if err != nil {
		p.log.Error(op, err)
		return err
	}
	// Проверка на то что строка была обновлена:
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		p.log.Error(op, err)
		return err
	}
	if rowsAffected == 0 {
		return ErrUserAlreadyInvited //Если cтрока не была изменена, значит поле invited_by уже было заполнено
	}
	p.log.Debug(fmt.Sprintf("%v: successfully set invited_by for user %v to %v", op, userID, invitedByID))
	return nil
}
