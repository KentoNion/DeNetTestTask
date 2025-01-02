package server

import (
	"app/domain"
	"time"
)

type user struct {
	id         domain.UserID    `json:"id"`
	nickname   domain.Nickname  `json:"nickname"`
	email      domain.Email     `json:"email,omitempty"` //omitempty чтоб не палить почту в leaderboard
	score      domain.UserScore `json:"score"`
	registered time.Time        `json:"register_date"`
	invitedBy  domain.UserID    `json:"invited_by,omitempty"` //omitempty потому что поле может быть пустым + ни к чему в leaderboard
}

func (u *user) toDomain() domain.User {
	return domain.User{
		Id:         u.id,
		Nickname:   u.nickname,
		Email:      u.email,
		Score:      u.score,
		Registered: u.registered,
		InvitedBy:  u.invitedBy,
	}
}

func fromDomain(duser domain.User) user {
	return user{
		id:         duser.Id,
		nickname:   duser.Nickname,
		email:      duser.Email,
		score:      duser.Score,
		registered: duser.Registered,
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
