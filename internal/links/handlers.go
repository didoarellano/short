package links

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/didoarellano/short/internal/auth"
	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/session"
	"github.com/didoarellano/short/internal/shortcode"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"
)

const paginationLimit int = 2

type PaginationLink struct {
	Href     string
	Text     string
	Disabled bool
}

type PaginationLinks []PaginationLink

func UserLinksHandler(t *template.Template, queries *db.Queries, sessionStore session.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "session")
		user := session.Values["user"].(auth.UserSession)
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

func CreateLinkHandler(t *template.Template, queries *db.Queries, sessionStore session.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		user := session.Values["user"].(auth.UserSession)
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
}

func UserLinkHandler(t *template.Template, queries *db.Queries, sessionStore session.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		session, _ := sessionStore.Get(r, "session")
		user := session.Values["user"].(auth.UserSession)
		userID := user.UserID

		link, err := queries.GetLinkForUser(context.Background(), db.GetLinkForUserParams{
			UserID:    userID,
			ShortCode: vars["shortcode"],
		})

		if err != nil {
			log.Printf("Failed to retrieve link: %v", err)
			http.Error(w, "Failed to retrieve link: %v", http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"user":       user,
			"link":       link,
			"wasUpdated": !link.CreatedAt.Time.Equal(link.UpdatedAt.Time),
		}

		if err := t.ExecuteTemplate(w, "link.html", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
