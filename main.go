package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/didoarellano/short/db"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/rbcervilla/redisstore/v8"
)

//go:embed templates/*
var resources embed.FS
var t = template.Must(template.ParseFS(resources, "templates/*"))

var queries *db.Queries
var sessionStore *redisstore.RedisStore

type ContextKey string

const UserContextKey ContextKey = "user"

func authSessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "auth")
		ctx := context.WithValue(r.Context(), UserContextKey, session.Values)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func oAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, "auth")
	if session.Values["id"] != nil {
		// user is already logged in
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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

	session.Values["id"] = user.ID
	session.Values["name"] = user.Name.String
	err = session.Save(r, w)

	if err != nil {
		log.Fatal(err)
		http.Error(w, "Failed to set session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func privateRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "auth")
		if session.Values["id"] == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func signinHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey)
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	if err := t.ExecuteTemplate(w, "signin.html", user); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "auth")
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

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey)
	if err := t.ExecuteTemplate(w, "dashboard.html", user); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func main() {
	ctx := context.Background()

	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal(err)
	}
	redis_client := redis.NewClient(opt)

	sessionStore, err = redisstore.NewRedisStore(ctx, redis_client)
	if err != nil {
		log.Fatal("Faled to create redis store", err)
	}

	pg_conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pg_conn.Close(ctx)
	queries = db.New(pg_conn)

	goth.UseProviders(
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			os.Getenv("GOOGLE_REDIRECT_URL"),
		),
	)

	r := mux.NewRouter()
	r.Use(authSessionMiddleware)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"greeting": "Hello world",
		}
		if err := t.ExecuteTemplate(w, "index.html", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}).Methods("GET")
	r.HandleFunc("/signin", signinHandler).Methods("GET")
	r.HandleFunc("/logout", logoutHandler).Methods("POST")
	r.HandleFunc("/auth/{provider}", gothic.BeginAuthHandler).Methods("GET")
	r.HandleFunc("/auth/{provider}/callback", oAuthCallbackHandler).Methods("GET")

	// Private routes
	r.Handle("/dashboard", privateRoute(http.HandlerFunc(dashboardHandler))).Methods("GET")

	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8080"
	}
	log.Println("Server started on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
