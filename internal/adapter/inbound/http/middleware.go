package http

import (
	"context"
	"net/http"
	"strings"

	"bakery/internal/app"
	"bakery/internal/domain/user"
)

type ctxKey string

const claimsKey ctxKey = "auth.claims"

// AuthMiddleware парсит Bearer-токен и кладёт claims в context.
// Если токена нет — пропускает публичные маршруты; защиту делает RequireAuth.
func AuthMiddleware(auth *app.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token := strings.TrimPrefix(header, "Bearer ")
				claims, err := auth.Verify(token)
				if err == nil {
					ctx := context.WithValue(r.Context(), claimsKey, claims)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth отбрасывает запрос без валидного токена.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ClaimsFromContext(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

// RequireRole требует одну из перечисленных ролей.
func RequireRole(roles ...user.Role) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			c, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			for _, role := range roles {
				if c.Role == role {
					next(w, r)
					return
				}
			}
			writeError(w, http.StatusForbidden, "forbidden")
		}
	}
}

func ClaimsFromContext(ctx context.Context) (app.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(app.Claims)
	return c, ok
}
