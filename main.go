package main

import (
	"context"
	"embed"
	"encoding/gob"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/shortcode"
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

type UserSession struct {
	UserID   int32
	Username string
}

func oAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, "session")
	if err != nil {
		log.Printf("Error retrieving session: %v", err)
		http.Error(w, "Failed to retrieve session", http.StatusInternalServerError)
		return
	}

	if session.Values["user"] != nil {
		// user is already logged in
		http.Redirect(w, r, "/links", http.StatusSeeOther)
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

	session.Values["user"] = UserSession{
		UserID:   user.ID,
		Username: user.Name.String,
	}
	err = session.Save(r, w)

	if err != nil {
		log.Fatal(err)
		http.Error(w, "Failed to set session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/links", http.StatusSeeOther)
}

func privateRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "session")
		user := session.Values["user"]
		if user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func signinHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "session")
	user := session.Values["user"]
	if user != nil {
		http.Redirect(w, r, "/links", http.StatusFound)
		return
	}
	if err := t.ExecuteTemplate(w, "signin.html", nil); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "session")
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

const paginationLimit int = 2

type PaginationLink struct {
	Href     string
	Text     string
	Disabled bool
}
type PaginationLinks []PaginationLink

func userLinksHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "session")
	user := session.Values["user"].(UserSession)
	userID := user.UserID

	// No page query param defaults to page 1
	currentPage := 1
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil {
			currentPage = parsedPage
		}
	}

	if currentPage < 1 {
		http.Redirect(w, r, "/links", http.StatusSeeOther)
		return
	}

	links, err := queries.GetPaginatedLinksForUser(context.Background(), db.GetPaginatedLinksForUserParams{
		UserID: userID,
		Limit:  int32(paginationLimit),
		Offset: int32((currentPage - 1) * paginationLimit),
	})

	if err != nil {
		log.Printf("Failed to retrieve user's links: %v", err)
		http.Error(w, "Failed to retrieve user's links: %v", http.StatusInternalServerError)
		return
	}

	totalPages := (int(links.TotalCount) + paginationLimit - 1) / paginationLimit

	if currentPage > totalPages {
		http.Redirect(w, r, fmt.Sprintf("/links?page=%d", totalPages), http.StatusSeeOther)
		return
	}

	paginationLinks := PaginationLinks{
		{
			Text:     "first",
			Href:     "/links",
			Disabled: currentPage == 1,
		},
		{
			Text:     "prev",
			Href:     fmt.Sprintf("/links?page=%d", currentPage-1),
			Disabled: currentPage == 1,
		},
		{
			Text:     "next",
			Href:     fmt.Sprintf("/links?page=%d", currentPage+1),
			Disabled: currentPage == totalPages,
		},
		{
			Text:     "last",
			Href:     fmt.Sprintf("/links?page=%d", totalPages),
			Disabled: currentPage == totalPages,
		},
	}

	data := map[string]interface{}{
		"user":            user,
		"links":           links.Links,
		"paginationLinks": paginationLinks,
	}

	if err := t.ExecuteTemplate(w, "links.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

type FormData struct {
	DestinationUrl  string
	Title           string
	Notes           string
	CreateDuplicate bool
}

type DuplicateUrl struct {
	Text string
	Href string
}

type DuplicateUrls struct {
	Urls           []DuplicateUrl
	DestinationUrl string
	RemainingCount int32
}

type FormFieldValidation struct {
	Message   string
	Value     string
	IsChecked bool
}

type FormValidationErrors struct {
	FormFields map[string]FormFieldValidation
	Duplicates DuplicateUrls
}

func CreateLinkHandler(w http.ResponseWriter, r *http.Request) {
	var validationErrors FormValidationErrors
	session, _ := sessionStore.Get(r, "session")

	if r.Method == "GET" {
		flashes := session.Flashes()
		if len(flashes) > 0 {
			if v, ok := flashes[0].(FormValidationErrors); ok {
				validationErrors = v
			}
		}
		data := map[string]interface{}{
			"validationErrors": validationErrors,
		}
		session.Save(r, w)
		if err := t.ExecuteTemplate(w, "create_link.html", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	user := session.Values["user"].(UserSession)
	userID := user.UserID

	r.ParseForm()
	formData := FormData{
		DestinationUrl:  strings.TrimSpace(r.FormValue("url")),
		Title:           strings.TrimSpace(r.FormValue("title")),
		Notes:           strings.TrimSpace(r.FormValue("notes")),
		CreateDuplicate: r.FormValue("create-duplicate") == "on",
	}

	if formData.DestinationUrl == "" {
		validationErrors := FormValidationErrors{
			FormFields: map[string]FormFieldValidation{
				"Url": {
					Value:   formData.DestinationUrl,
					Message: "Destination URL is required",
				},
				"Title": {
					Value: formData.Title,
				},
				"Notes": {
					Value: formData.Notes,
				},
				"CreateDuplicate": {
					IsChecked: formData.CreateDuplicate,
				},
			},
		}
		session.AddFlash(validationErrors)
		session.Save(r, w)
		http.Redirect(w, r, "/links/new", http.StatusFound)
		return
	}

	if !formData.CreateDuplicate {
		links, _ := queries.FindDuplicatesForUrl(context.Background(), db.FindDuplicatesForUrlParams{
			UserID:         userID,
			DestinationUrl: formData.DestinationUrl,
			Limit:          3,
		})

		if len(links.ShortCodes) > 0 {
			dupes := DuplicateUrls{
				DestinationUrl: formData.DestinationUrl,
				RemainingCount: links.RemainingCount,
			}
			for _, shortcode := range links.ShortCodes {
				url := DuplicateUrl{
					Href: "/links/" + shortcode,
					Text: shortcode,
				}
				dupes.Urls = append(dupes.Urls, url)
			}
			validationErrors := FormValidationErrors{
				Duplicates: dupes,
				FormFields: map[string]FormFieldValidation{
					"Url": {
						Value: formData.DestinationUrl,
					},
					"Title": {
						Value: formData.Title,
					},
					"Notes": {
						Value: formData.Notes,
					},
					"CreateDuplicate": {
						IsChecked: formData.CreateDuplicate,
					},
				},
			}
			session.AddFlash(validationErrors)
			session.Save(r, w)
			http.Redirect(w, r, "/links/new", http.StatusSeeOther)
			return
		}
	}

	shortCode := shortcode.New(userID, formData.DestinationUrl, 7)

	_, err := queries.CreateLink(context.Background(), db.CreateLinkParams{
		UserID:         userID,
		ShortCode:      shortCode,
		DestinationUrl: formData.DestinationUrl,
		Title:          pgtype.Text{String: formData.Title, Valid: true},
		Notes:          pgtype.Text{String: formData.Notes, Valid: true},
	})

	if err != nil {
		log.Printf("Failed to create new link: %v", err)
		http.Error(w, "Failed to create new link", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/links", http.StatusSeeOther)
}

func main() {
	gob.Register(UserSession{})
	gob.Register(FormValidationErrors{})
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
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := t.ExecuteTemplate(w, "404.html", nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	})

	// Private routes
	r.Handle("/links", privateRoute(http.HandlerFunc(userLinksHandler))).Methods("GET")
	r.Handle("/links/new", privateRoute(http.HandlerFunc(CreateLinkHandler))).Methods("GET", "POST")

	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8080"
	}
	log.Println("Server started on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
