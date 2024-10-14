package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/didoarellano/short/internal/config"
	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/session"
	"github.com/didoarellano/short/internal/templ"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/markbates/goth/gothic"
)

type AuthHandler struct {
	template     *templ.Templ
	queries      *db.Queries
	sessionStore session.SessionStore
}

func NewAuthHandlers(t *templ.Templ, q *db.Queries, s session.SessionStore) *AuthHandler {
	return &AuthHandler{
		template:     t,
		queries:      q,
		sessionStore: s,
	}
}

func (ah *AuthHandler) BeginAuth(w http.ResponseWriter, r *http.Request) {
	gothic.BeginAuthHandler(w, r)
}

type UserSession struct {
	UserID   int32
	Username string
}

func (ah *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	session, err := ah.sessionStore.Get(r, "session")
	if err != nil {
		log.Printf("Error retrieving session: %v", err)
		http.Error(w, "Failed to retrieve session", http.StatusInternalServerError)
		return
	}

	if session.Values["user"] != nil {
		// user is already logged in
		http.Redirect(w, r, "/"+config.AppData.AppPathPrefix+"/links", http.StatusSeeOther)
		return
	}

	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	user, err := ah.queries.GetUserByEmail(ctx, gothUser.Email)
	if err == pgx.ErrNoRows {
		newUser, err := ah.queries.CreateUser(ctx, db.CreateUserParams{
			Name:          pgtype.Text{String: gothUser.NickName, Valid: gothUser.NickName != ""},
			Email:         gothUser.Email,
			OauthProvider: pgtype.Text{String: gothUser.Provider, Valid: gothUser.Provider != ""},
			Role:          "basic",
		})

		if err != nil {
			log.Printf("Failed to create user: %v", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		// Workaround for sqlc not generating a shared type for GetUserByEmail and CreateUser queries.
		// Make sure the two queries in queries.sql always return the same columns.
		user = db.GetUserByEmailRow(newUser)
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

	http.Redirect(w, r, "/"+config.AppData.AppPathPrefix+"/links", http.StatusSeeOther)
}

func (ah *AuthHandler) Signin(w http.ResponseWriter, r *http.Request) {
	session, _ := ah.sessionStore.Get(r, "session")
	user := session.Values["user"]
	if user != nil {
		http.Redirect(w, r, "/"+config.AppData.AppPathPrefix+"/links", http.StatusSeeOther)
		return
	}
	if err := ah.template.ExecuteTemplate(w, "signin.html", nil); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (ah *AuthHandler) Signout(w http.ResponseWriter, r *http.Request) {
	session, _ := ah.sessionStore.Get(r, "session")
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
