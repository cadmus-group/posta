package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
)

type Authenticator struct {
	token string
}

func newAuthenticator(log *slog.Logger, token string) *Authenticator {
	token = strings.TrimSpace(token)
	if token == "" {
		log.Warn("enqueue auth disabled; set ENQUEUE_AUTH_TOKEN to require tokens on /v1/enqueue")
		return nil
	}

	return &Authenticator{token: token}
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	if a == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.isAuthorized(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Authenticator) isAuthorized(r *http.Request) bool {
	if a == nil {
		return true
	}

	token := strings.TrimSpace(r.Header.Get("X-Edge-Token"))
	if token == "" {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		const bearerPrefix = "bearer "
		if len(authHeader) > len(bearerPrefix) && strings.EqualFold(authHeader[:len(bearerPrefix)], bearerPrefix) {
			token = strings.TrimSpace(authHeader[len(bearerPrefix):])
		}
	}

	return token != "" && token == a.token
}
