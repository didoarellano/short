package redirector

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/didoarellano/short/internal/db"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

func RedirectHandler(q *db.Queries, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		shortcode := vars["shortcode"]

		ctx := context.Background()
		key := fmt.Sprintf("shortcode:%s", shortcode)
		destinationUrl, err := redisClient.Get(ctx, key).Result()

		if err == redis.Nil {
			destinationUrl, err = q.GetDestinationUrl(ctx, shortcode)
			if err != nil {
				log.Printf("Destination URL not found: %v", err)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			err = redisClient.Set(ctx, key, destinationUrl, 24*time.Hour).Err()
			if err != nil {
				log.Printf("Failed to cache shortcode: %v", err)
			}
		} else if err != nil {
			log.Printf("redis error %v:", err)
		}

		log.Println("Redirecting to", destinationUrl)
		http.Redirect(w, r, destinationUrl, http.StatusSeeOther)
	}
}
