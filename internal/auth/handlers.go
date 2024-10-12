package auth

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/session"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/markbates/goth/gothic"
)

type UserSession struct {
	UserID   int32
	Username string
}

func OAuthCallbackHandler(queries *db.Queries, sessionStore session.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, "session")
		if err != nil {
			log.Printf("Error retrieving session: %v", err)
			http.Error(w, "Failed to retrieve session", http.StatusInternalServerError)
			return
		}

		if session.Values["user"] != nil {
			// user is already logged in
			http.Redirect(w, r, "/links", http.StatusSeeOther)
			return
		}

		gothUser, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		user, err := queries.CreateOrUpdateUser(context.Background(), db.CreateOrUpdateUserParams{
			Name:          pgtype.Text{String: gothUser.NickName, Valid: gothUser.NickName != ""},
			Email:         gothUser.Email,
			OauthProvider: pgtype.Text{String: gothUser.Provider, Valid: gothUser.Provider != ""},
			Role:          "basic",
		})

		if err != nil {
			log.Printf("Failed to create or update user: %v", err)
			http.Error(w, "Failed to create or update user", http.StatusInternalServerError)
			return
		}

		session.Values["user"] = UserSession{
			UserID:   user.ID,
			Username: user.Name.String,
		}
		err = session.Save(r, w)

		if err != nil {
			log.Fatal(err)
			http.Error(w, "Failed to set session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/links", http.StatusSeeOther)
	}
}

func SigninHandler(t *template.Template, sessionStore session.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "session")
		user := session.Values["user"]
		if user != nil {
			http.Redirect(w, r, "/links", http.StatusFound)
			return
		}
		if err := t.ExecuteTemplate(w, "signin.html", nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}

func SignoutHandler(sessionStore session.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "session")
		session.Options.MaxAge = -1
		err := session.Save(r, w)

		if err != nil {
			log.Fatal("Failed to delete session", err)
			return
		}

		err = gothic.Logout(w, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Logout failed: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
