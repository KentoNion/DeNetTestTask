package server

import (
	"app/domain"
	"context"
	"fmt"
	"net/http"
	"strings"
)

func (s Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "gates.server.authMiddleware"
		s.log.Info(op, ": starting auth")
		authHeader := r.Header.Get("Authorization")
		s.log.Debug("auth header: ", authHeader)
		if authHeader == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			s.log.Debug(op, ": no auth header")
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			s.log.Debug(op, ": invalid auth header format")
			http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		token := parts[1]
		// Проверяем токен через auth.Authorize
		s.log.Debug(fmt.Sprintf("%s: trying to get token thru auth.Authorize with token: %s", op, token))
		user, err := s.auth.Authorize(s.context, token)
		if err != nil {
			http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		s.log.Debug(op, ": Successfully got token thru auth.Authorize for user: ", user.ID)
		// Добавляем пользователя в контекст
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext - извлекает пользователя из контекста
func userFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(domain.User)
	return user, ok
}
