package auth

import (
	"net/http"

	"github.com/didoarellano/short/internal/session"
)

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
