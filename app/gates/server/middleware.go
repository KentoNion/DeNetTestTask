package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (s Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "gates.server.authMiddleware"
		s.log.Info(op, ": starting auth")

		authHeader := c.GetHeader("Authorization")
		s.log.Debug("auth header: ", authHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			s.log.Debug(op, ": no auth header")
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			s.log.Debug(op, ": invalid auth header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header format"})
			return
		}
		token := parts[1]
		// Проверяем токен через auth.Authorize
		s.log.Debug(fmt.Sprintf("%s: trying to get token thru auth.Authorize with token: %s", op, token))
		user, err := s.auth.Authorize(s.context, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: " + err.Error()})
			return
		}
		s.log.Debug(op, ": Successfully got token thru auth.Authorize for user: ", user.ID)
		// Добавляем пользователя в контекст
		c.Set("user", user)
		c.Next()
	}
}
