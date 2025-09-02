package middleware

import (
	"context"
	"net/http"

	"github.com/rudraprasaaad/task-scheduler/internal/auth"
)

type contextKey string

const UserContextkey = contextKey("user")

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
				return
			}

			tokenStr := cookie.Value
			claims, err := auth.ValidateToken(tokenStr, jwtSecret)
			if err != nil {
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextkey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
