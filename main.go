package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
	_ "github.com/joho/godotenv/autoload"
)

//go:embed templates/*
var resources embed.FS
var t = template.Must(template.ParseFS(resources, "templates/*"))

func main() {
	opt, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	rdb := redis.NewClient(opt)

	pg_conn, err := pgx.Connect(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pg_conn.Close(context.Background())

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	var greeting string
	err = pg_conn.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"greeting": greeting,
		}
		if err := t.ExecuteTemplate(w, "index.html", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	})

	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
