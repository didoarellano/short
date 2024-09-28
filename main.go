package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/rbcervilla/redisstore/v8"
)

//go:embed templates/*
var resources embed.FS
var t = template.Must(template.ParseFS(resources, "templates/*"))

var sessionStore *redisstore.RedisStore

func oAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO Save user to db

	session, _ := sessionStore.Get(r, "auth")
	session.Values["id"] = gothUser.UserID
	session.Values["username"] = gothUser.NickName
	err = session.Save(r, w)

	if err != nil {
		log.Fatal(err)
		http.Error(w, "Failed to set session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func signinHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "auth")
	if session.Values["id"] != nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	if err := t.ExecuteTemplate(w, "signin.html", nil); err != nil {
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

func main() {
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal(err)
	}

	redis_client := redis.NewClient(opt)

	sessionStore, err = redisstore.NewRedisStore(context.Background(), redis_client)
	if err != nil {
		log.Fatal("Faled to create redis store", err)
	}

	goth.UseProviders(
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			os.Getenv("GOOGLE_REDIRECT_URL"),
		),
	)

	r := mux.NewRouter()
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

	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8080"
	}
	log.Println("Server started on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
