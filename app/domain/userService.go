package domain

import (
	"app/iternal/config"
	"context"
	"fmt"
	"log/slog"
)

type UserService struct {
	store UserStore
	log   *slog.Logger
	cfg   *config.Config
}

type UserStore interface {
	GetUser(ctx context.Context, id UserID) (User, error)
	GetUsers(ctx context.Context, filter Sorter, page int, limit int) ([]User, error)
	AddPoints(ctx context.Context, id UserID, points int) error
	SetInvitedBy(ctx context.Context, userID UserID, invitedByID UserID) error
	AddUser(ctx context.Context, user User) error
}

func NewUserService(store UserStore, log *slog.Logger, cfg *config.Config) *UserService {
	return &UserService{
		store: store,
		log:   log,
		cfg:   cfg,
	}
}

func (s UserService) AddUser(ctx context.Context, user User) error {
	const op = "UserService.AddUser"
	s.log.Debug(op, "trying to add user")
	err := s.store.AddUser(ctx, user)
	if err != nil {
		return err
	}
	s.log.Debug(op, "successfully added user")
	return nil
}

func (s UserService) Status(ctx context.Context, id UserID) (User, error) {
	const op = "UserService.Status"
	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		s.log.Error(op, err)
		return User{}, err
	}
	return user, err
}

func (s UserService) Leaderbord(ctx context.Context, filter Sorter, page int, limit int) ([]User, error) {
	const op = "UserService.Leaderbord"

	users, err := s.store.GetUsers(ctx, filter, page, limit)
	if err != nil {
		s.log.Error(op, err)
		return nil, err
	}
	return users, err
}

func (s UserService) TaskComplete(ctx context.Context, id UserID, task string) error {
	const op = "UserService.TaskComplete"
	var err error
	if points, inMap := s.cfg.Rewards[task]; inMap {
		err = s.store.AddPoints(ctx, id, points)
		if err != nil {
			return err
		}
	} else {
		s.log.Info(op, fmt.Sprintf("user %v tried to claim not existing reward", id))
		return ErrNotExistingReward
	}
	return nil
}

func (s UserService) InvitedBy(ctx context.Context, id UserID, invitedBy UserID) error {
	const op = "UserService.InvitedBy"
	rewardInviter := s.cfg.Rewards["inviting_a_friend"]
	rewardInvited := s.cfg.Rewards["being_invited"]
	if rewardInviter == 0 {
		s.log.Error("No reward for ref")
		return ErrNoRewardRef
	}
	err := s.store.SetInvitedBy(ctx, id, invitedBy)
	if err != nil {
		return err
	}
	err = s.store.AddPoints(ctx, id, rewardInviter)
	if err != nil {
		return err
	}
	err = s.store.AddPoints(ctx, invitedBy, rewardInvited)
	if err != nil {
		return err
	}
	//todo: По хорошему следовало бы объеденить все эти 3 запроса в одну транзакцию для снижения кол-во запросов к бд и атомарности
	return nil
}
