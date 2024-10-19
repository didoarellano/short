package redirector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/didoarellano/short/internal/db"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mileusna/useragent"
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

		uaData, _ := parseUserAgent(r.UserAgent())
		referrer := r.Referer()
		q.RecordVisit(ctx, db.RecordVisitParams{
			ShortCode:     shortcode,
			UserAgentData: uaData,
			ReferrerUrl:   pgtype.Text{String: referrer, Valid: referrer != ""},
		})

		log.Println("Redirecting to", destinationUrl)
		http.Redirect(w, r, destinationUrl, http.StatusSeeOther)
	}
}

type UserAgentDetails struct {
	UAString       string `json:"ua_string"`
	BrowserName    string `json:"browser_name"`
	BrowserVersion string `json:"browser_version"`
	OSName         string `json:"os_name"`
	OSVersion      string `json:"os_version"`
	Device         string `json:"device"`
	Type           string `json:"type"`
}

func parseUserAgent(uaString string) ([]byte, error) {
	ua := useragent.Parse(uaString)

	details := UserAgentDetails{
		UAString:       uaString,
		BrowserName:    ua.Name,
		BrowserVersion: ua.Version,
		OSName:         ua.OS,
		OSVersion:      ua.OSVersion,
		Device:         ua.Device,
	}

	switch {
	case ua.Mobile:
		details.Type = "Mobile"
	case ua.Tablet:
		details.Type = "Tablet"
	case ua.Desktop:
		details.Type = "Desktop"
	case ua.Bot:
		details.Type = "Bot"
	default:
		details.Type = "Unknown"
	}

	uaJSON, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user agent details: %w", err)
	}

	return uaJSON, nil
}
