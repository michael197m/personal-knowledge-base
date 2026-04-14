package http

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"personal-knowledge-base/backend/internal/store"
)

type tokenVerifier interface {
	ParseToken(token string) (uuid.UUID, error)
}

func withAuth(verifier tokenVerifier, cookieName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const bearerPrefix = "Bearer "

		var token string
		if cookie, err := r.Cookie(cookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
			token = strings.TrimSpace(cookie.Value)
		}

		if token == "" {
			authorizationHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(authorizationHeader, bearerPrefix) {
				writeUnauthorized(w)
				return
			}

			token = strings.TrimSpace(strings.TrimPrefix(authorizationHeader, bearerPrefix))
		}
		if token == "" {
			writeUnauthorized(w)
			return
		}

		userID, err := verifier.ParseToken(token)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		ctx := store.ContextWithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"authentication required"}`))
}
