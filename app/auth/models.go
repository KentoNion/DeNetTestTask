package auth

import (
	"app/domain"
	"app/iternal/pkg"
	"errors"
	jwt "github.com/golang-jwt/jwt/v5"
	"time"
)

type Token struct {
	UserID   domain.UserID
	Email    domain.Email
	Nickname domain.Nickname
}

func (t Token) MapToAccess(cl pkg.Clock, ttl time.Duration) jwt.Claims {
	return jwt.MapClaims{
		"user_id":  t.UserID,
		"email":    t.Email,
		"nickname": t.Nickname,
		"exp":      cl.Now().Add(ttl).Unix(),
	}
}

var ErrMismatchTokenData = errors.New("token data doesn't match db data")
