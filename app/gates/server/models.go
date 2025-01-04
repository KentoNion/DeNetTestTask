package server

import (
	"app/domain"
	"time"
)

type user struct {
	Id         domain.UserID    `json:"Id"`
	Nickname   domain.Nickname  `json:"Nickname"`
	Email      domain.Email     `json:"Email,omitempty"` //omitempty чтоб не палить почту в leaderboard
	Score      domain.UserScore `json:"Score"`
	Registered time.Time        `json:"register_date"`
	invitedBy  *domain.UserID   `json:"invited_by,omitempty"` //omitempty потому что поле может быть пустым + ни к чему в leaderboard
}

func (u *user) toDomain() domain.User {
	return domain.User{
		ID:         u.Id,
		Nickname:   u.Nickname,
		Email:      u.Email,
		Score:      u.Score,
		Registered: u.Registered,
		InvitedBy:  u.invitedBy,
	}
}

func fromDomain(duser domain.User) user {
	return user{
		Id:         duser.ID,
		Nickname:   duser.Nickname,
		Email:      duser.Email,
		Score:      duser.Score,
		Registered: duser.Registered,
		invitedBy:  duser.InvitedBy,
	}
}

type contextKey string

const userContextKey contextKey = "user"

type leaderboardSettings struct {
	sorter domain.Sorter `json:"sort_by"`
	page   int           `json:"page"`
	size   int           `json:"size"`
}
