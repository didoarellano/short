package auth

import (
	"context"
	"net/http"

	"github.com/didoarellano/short/internal/session"
)

type key string

const SubscriptionKey key = "subscription"

func (ah *AuthHandler) UserSubscriptionMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, _ := ah.sessionStore.Get(r, "session")
			user := session.Values["user"].(UserSession)
			userID := user.UserID
			subscription, _ := ah.GetSubscriptionForUser(userID)
			ctx := context.WithValue(r.Context(), SubscriptionKey, subscription)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func PrivateRoute(sessionStore session.SessionStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, _ := sessionStore.Get(r, "session")
			user := session.Values["user"]
			if user == nil {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
