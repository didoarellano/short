package main

import (
	"context"
	"embed"
	"encoding/gob"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/didoarellano/short/internal/auth"
	"github.com/didoarellano/short/internal/config"
	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/geodata"
	"github.com/didoarellano/short/internal/links"
	"github.com/didoarellano/short/internal/redirector"
	"github.com/didoarellano/short/internal/subscriptions"
	"github.com/didoarellano/short/internal/templ"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbcervilla/redisstore/v8"
)

//go:embed templates/*
var resources embed.FS
var stdtemplate = template.Must(template.ParseFS(resources, "templates/*"))

//go:embed static/*
var static embed.FS

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

	dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer dbpool.Close()
	queries = db.New(dbpool)

	auth.Initialise()
	t := templ.New(stdtemplate, config.AppData, sessionStore)

	rootRouter := mux.NewRouter()

	var geodataFetcher geodata.GeoDataFetcher
	if os.Getenv("ENV") == "dev" {
		geodataFetcher = &geodata.MockGeoDataFetcher{}
	} else {
		geodataFetcher = &geodata.RealGeoDataFetcher{}
	}

	redirector := redirector.New(queries, redisClient, geodataFetcher)
	rootRouter.HandleFunc("/{shortcode}", redirector.RedirectHandler).Methods("GET")

	rootRouter.HandleFunc("/", t.RenderStatic("index.html")).Methods("GET")
	rootRouter.NotFoundHandler = t.RenderStatic("404.html")

	authHandlers := auth.NewAuthHandlers(t, queries, sessionStore, redisClient)
	appRouter := rootRouter.PathPrefix("/" + config.AppData.AppPathPrefix).Subrouter()

	subFS, _ := fs.Sub(static, "static")
	fs := http.FileServer(http.FS(subFS))
	appRouter.PathPrefix("/static/").Handler(http.StripPrefix("/"+config.AppData.AppPathPrefix+"/static/", fs))

	appRouter.HandleFunc("/signin", authHandlers.Signin).Methods("GET")
	appRouter.HandleFunc("/signout", authHandlers.Signout).Methods("POST")
	appRouter.HandleFunc("/auth/{provider}", authHandlers.BeginAuth).Methods("GET")
	appRouter.HandleFunc("/auth/{provider}/callback", authHandlers.OAuthCallback).Methods("GET")

	userSubscriptionService := subscriptions.NewUserSubscriptionService(queries, sessionStore, redisClient)
	linkHandlers := links.NewLinkHandlers(t, queries, sessionStore, redisClient, *userSubscriptionService)
	privateAppRouter := appRouter.PathPrefix("/").Subrouter()
	privateAppRouter.Use(auth.PrivateRoute(sessionStore))
	privateAppRouter.Use(userSubscriptionService.UserSubscriptionMiddleware())
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
