package auth

/*Данный пакет моковый, в тестовом задании ничего не сказано про то как должна осуществляться
аунтефикакация, я просто написал моковую функцию login которая генерирует jwt access токен
и функцию auth которая исплоьзуется в middleware, которая просто разбирает jwt и сверяет данныс с бд
и проверяет не протух ли токен (не вышло ли его время)
У меня есть другое тестовое задание с нормальной реализацией аунтефикации, ознакомиться можно здесь:
https://github.com/KentoNion/medodsTestovoeAuth
*/
import (
	"app/domain"
	"app/iternal/config"
	"app/iternal/pkg"
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"time"
)

type Service struct {
	secretKey string
	store     domain.UserStore
	log       *slog.Logger
	cfg       *config.Config
	cl        pkg.Clock
}

func NewService(store domain.UserStore, log *slog.Logger, cfg *config.Config, secret string, cl pkg.Clock) *Service {
	return &Service{
		secretKey: secret,
		store:     store,
		log:       log,
		cfg:       cfg,
		cl:        cl,
	}
}

func (s *Service) Login(ctx context.Context, id domain.UserID) (string, error) {
	const op = "auth.Login"
	s.log.Debug(op, ": Starting login process for user", id)

	// Извлекаем пользователя из бд (и проверяем есть ли он там)
	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		s.log.Error(op, "Failed to check user existence", err)
		return "", err
	}
	token := Token{
		UserID:   user.Id,
		Email:    user.Email,
		Nickname: user.Nickname,
	}
	claims := token.MapToAccess(s.cl, time.Hour) // TTL = 1 час

	signedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := signedToken.SignedString([]byte(s.secretKey))
	if err != nil {
		s.log.Error("Failed to generate JWT", "op", op, "error", err)
		return "", err
	}

	s.log.Debug(op, ": Access generated successfully")
	return tokenString, nil
}

func (s *Service) Authorize(ctx context.Context, accessToken string) (domain.User, error) {
	const op = "auth.Authorize"
	var user domain.User
	s.log.Debug(op, ": trying to authorize user")

	token, err := jwt.Parse(accessToken, func(t *jwt.Token) (interface{}, error) {

		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			s.log.Warn("Unexpected signing method", "op", op, "method", t.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})
	if err != nil {
		s.log.Error("Failed to parse token", "op", op, "error", err)
		return user, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		s.log.Warn("Invalid token claims", "op", op)
		return user, fmt.Errorf("invalid token claims")
	}
	//Проверяем не истекло ли время жизни токена
	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			s.log.Warn("Token expired", "op", op)
			return user, fmt.Errorf("token has expired")
		}
	} else {
		s.log.Warn("Token expiration missing", "op", op)
		return user, fmt.Errorf("token expiration missing")
	}

	// Извлекаем данные из токена
	userID, ok := claims["user_id"].(string)
	if !ok {
		s.log.Warn("User ID missing in token", "op", op)
		return user, fmt.Errorf("user ID missing in token")
	}
	//Проверяем наличие эмеила
	email, ok := claims["email"].(string)
	if !ok {
		s.log.Warn(op, ": Email is missing in token")
		return user, fmt.Errorf("email missing in token")
	}
	//проверяем наличие никнейма
	nickname, ok := claims["nickname"].(string)
	if !ok {
		s.log.Warn(op, ": Nickname is missing in token")
		return user, fmt.Errorf("nickname missing in token")
	}

	// Вытаскиваем данные пользователя из бд
	user, err = s.store.GetUser(ctx, domain.UserID(userID))
	if err != nil {
		s.log.Error(op, "failed to retrieve user from db")
		return user, fmt.Errorf("failed to retrieve user: %w", err)
	}
	//сверяем эмеил
	if user.Email != domain.Email(email) {
		s.log.Warn(op, ": Token data does not match database")
		return user, ErrMismatchTokenData
	}
	//Сверяем никнейм
	if user.Nickname != domain.Nickname(nickname) {
		s.log.Warn(op, ": Token data does not match database")
		return user, ErrMismatchTokenData
	}

	s.log.Info("Authorization successful", "op", op, "user_id", userID)
	return user, nil
}
