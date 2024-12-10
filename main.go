package main

import (
	"context"
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
		DB:   0,
	})

	db, err := sql.Open("sqlite3", "short.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t := template.Must(template.ParseFiles("templates/index.html"))
		if err := t.Execute(w, nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
