package redirector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/geodata"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mileusna/useragent"
)

type Redirector struct {
	queries        *db.Queries
	redisClient    *redis.Client
	geodataFetcher geodata.GeoDataFetcher
}

func New(q *db.Queries, r *redis.Client, g geodata.GeoDataFetcher) *Redirector {
	return &Redirector{
		queries:        q,
		redisClient:    r,
		geodataFetcher: g,
	}
}

func (rr *Redirector) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortcode := vars["shortcode"]

	ctx := context.Background()
	key := fmt.Sprintf("shortcode:%s", shortcode)
	destinationUrl, err := rr.redisClient.Get(ctx, key).Result()

	if err == redis.Nil {
		destinationUrl, err = rr.queries.GetDestinationUrl(ctx, shortcode)
		if err != nil {
			log.Printf("Destination URL not found: %v", err)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		err = rr.redisClient.Set(ctx, key, destinationUrl, 24*time.Hour).Err()
		if err != nil {
			log.Printf("Failed to cache shortcode: %v", err)
		}
	} else if err != nil {
		log.Printf("redis error %v:", err)
	}

	go rr.RecordVisit(ctx, r, shortcode)

	http.Redirect(w, r, destinationUrl, http.StatusSeeOther)
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

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		clientIP := strings.TrimSpace(ips[0])
		return clientIP
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return ip
}

func (rr *Redirector) GetGeoData(r *http.Request) geodata.GeoData {
	ip := getClientIP(r)
	geoData, _ := rr.geodataFetcher.GetGeoData(net.ParseIP(ip))
	return geoData
}

func (rr *Redirector) RecordVisit(ctx context.Context, r *http.Request, shortcode string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in background job: %v", r)
		}
	}()

	uaData, _ := parseUserAgent(r.UserAgent())
	referrer := r.Referer()
	geoData := rr.GetGeoData(r)
	geoDataJSON, _ := json.Marshal(geoData)

	err := rr.queries.RecordVisit(ctx, db.RecordVisitParams{
		ShortCode:     shortcode,
		UserAgentData: uaData,
		GeoData:       geoDataJSON,
		ReferrerUrl:   pgtype.Text{String: referrer, Valid: referrer != ""},
	})

	if err != nil {
		log.Printf("Failed to record visit: %v", err)
	}
}
