package main

import (
	"context"
	"embed"
	"encoding/gob"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/didoarellano/short/internal/auth"
	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/links"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
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

func main() {
	gob.Register(auth.UserSession{})
	gob.Register(links.FormValidationErrors{})
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

	router := mux.NewRouter()
	router.HandleFunc("/", renderStatic("index.html")).Methods("GET")
	router.HandleFunc("/signin", auth.SigninHandler(t, sessionStore)).Methods("GET")
	router.HandleFunc("/signout", auth.SignoutHandler(sessionStore)).Methods("POST")
	router.HandleFunc("/auth/{provider}", gothic.BeginAuthHandler).Methods("GET")
	router.HandleFunc("/auth/{provider}/callback", auth.OAuthCallbackHandler(queries, sessionStore)).Methods("GET")
	router.NotFoundHandler = renderStatic("404.html")

	privateRouter := router.PathPrefix("/").Subrouter()
	privateRouter.Use(auth.PrivateRoute(sessionStore))

	privateRouter.HandleFunc("/links", links.UserLinksHandler(t, queries, sessionStore)).Methods("GET")
	privateRouter.HandleFunc("/links/new", links.CreateLinkHandler(t, queries, sessionStore)).Methods("GET", "POST")
	privateRouter.HandleFunc("/links/{shortcode}", links.UserLinkHandler(t, queries, sessionStore)).Methods("GET")

	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8080"
	}
	log.Println("Server started on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func renderStatic(template string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := t.ExecuteTemplate(w, template, nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
