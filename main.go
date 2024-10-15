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
	"github.com/didoarellano/short/internal/config"
	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/links"
	"github.com/didoarellano/short/internal/redirector"
	"github.com/didoarellano/short/internal/templ"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/rbcervilla/redisstore/v8"
)

//go:embed templates/*
var resources embed.FS
var stdtemplate = template.Must(template.ParseFS(resources, "templates/*"))

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
	redisClient := redis.NewClient(opt)

	sessionStore, err = redisstore.NewRedisStore(ctx, redisClient)
	if err != nil {
		log.Fatal("Faled to create redis store", err)
	}

	pg_conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pg_conn.Close(ctx)
	queries = db.New(pg_conn)

	auth.Initialise()
	t := templ.New(stdtemplate, config.AppData)

	rootRouter := mux.NewRouter()

	rootRouter.HandleFunc("/{shortcode}", redirector.RedirectHandler(queries, redisClient)).Methods("GET")

	rootRouter.HandleFunc("/", t.RenderStatic("index.html")).Methods("GET")
	rootRouter.NotFoundHandler = t.RenderStatic("404.html")

	authHandlers := auth.NewAuthHandlers(t, queries, sessionStore, redisClient)
	appRouter := rootRouter.PathPrefix("/" + config.AppData.AppPathPrefix).Subrouter()
	appRouter.HandleFunc("/signin", authHandlers.Signin).Methods("GET")
	appRouter.HandleFunc("/signout", authHandlers.Signout).Methods("POST")
	appRouter.HandleFunc("/auth/{provider}", authHandlers.BeginAuth).Methods("GET")
	appRouter.HandleFunc("/auth/{provider}/callback", authHandlers.OAuthCallback).Methods("GET")

	linkHandlers := links.NewLinkHandlers(t, queries, sessionStore, redisClient)
	privateAppRouter := appRouter.PathPrefix("/").Subrouter()
	privateAppRouter.Use(auth.PrivateRoute(sessionStore))
	privateAppRouter.HandleFunc("/links", linkHandlers.UserLinks).Methods("GET")
	privateAppRouter.HandleFunc("/links/new", linkHandlers.CreateLink).Methods("GET", "POST")
	privateAppRouter.HandleFunc("/links/{shortcode}", linkHandlers.UserLink).Methods("GET")

	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8080"
	}
	log.Println("Server started on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, rootRouter))
}
