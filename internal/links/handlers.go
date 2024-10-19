package links

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/didoarellano/short/internal/auth"
	"github.com/didoarellano/short/internal/config"
	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/session"
	"github.com/didoarellano/short/internal/subscriptions"
	"github.com/didoarellano/short/internal/templ"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

type LinkHandler struct {
	template     *templ.Templ
	queries      *db.Queries
	sessionStore session.SessionStore
	redisClient  *redis.Client
}

func NewLinkHandlers(t *templ.Templ, q *db.Queries, s session.SessionStore, r *redis.Client) *LinkHandler {
	return &LinkHandler{
		template:     t,
		queries:      q,
		sessionStore: s,
		redisClient:  r,
	}
}

const paginationLimit int = 2

type PaginationLink struct {
	Href     string
	Text     string
	Disabled bool
}

type PaginationLinks []PaginationLink

func (lh *LinkHandler) UserLinks(w http.ResponseWriter, r *http.Request) {
	session, _ := lh.sessionStore.Get(r, "session")
	user := session.Values["user"].(auth.UserSession)
	userID := user.UserID
	basePath := "/" + config.AppData.AppPathPrefix + "/links"

	// No page query param defaults to page 1
	currentPage := 1
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil {
			currentPage = parsedPage
		}
	}

	if currentPage < 1 {
		http.Redirect(w, r, basePath, http.StatusSeeOther)
		return
	}

	links, err := lh.queries.GetPaginatedLinksForUser(context.Background(), db.GetPaginatedLinksForUserParams{
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

	if totalPages > 0 && currentPage > totalPages {
		http.Redirect(w, r, fmt.Sprintf("%s?page=%d", basePath, totalPages), http.StatusSeeOther)
		return
	}

	paginationLinks := PaginationLinks{
		{
			Text:     "first",
			Href:     basePath,
			Disabled: currentPage == 1,
		},
		{
			Text:     "prev",
			Href:     fmt.Sprintf("%s?page=%d", basePath, currentPage-1),
			Disabled: currentPage == 1,
		},
		{
			Text:     "next",
			Href:     fmt.Sprintf("%s?page=%d", basePath, currentPage+1),
			Disabled: currentPage == totalPages,
		},
		{
			Text:     "last",
			Href:     fmt.Sprintf("%s?page=%d", basePath, totalPages),
			Disabled: currentPage == totalPages,
		},
	}

	data := map[string]interface{}{
		"user":            user,
		"links":           links.Links,
		"paginationLinks": paginationLinks,
	}

	if err := lh.template.ExecuteTemplate(w, "links.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (lh *LinkHandler) CreateLink(w http.ResponseWriter, r *http.Request) {
	session, _ := lh.sessionStore.Get(r, "session")
	basePath := "/" + config.AppData.AppPathPrefix + "/links"
	user := session.Values["user"].(auth.UserSession)
	userID := user.UserID

	userSubscriptionContext := r.Context().Value(subscriptions.SubscriptionKey).(subscriptions.UserSubscriptionContext)
	subscription := userSubscriptionContext.Subscription
	linksCreated := userSubscriptionContext.LinksCreated

	if r.Method == "GET" {
		ShowCreateForm(ShowCreateFormParams{
			w:                w,
			r:                r,
			session:          session,
			template:         lh.template,
			userSubscription: subscription,
			linksCreated:     linksCreated,
		})
		return
	}

	if linksCreated >= subscription.MaxLinksPerMonth {
		session.AddFlash(FormValidationErrors{
			Message: "You can't create anymore links this month. Upgrade to pro for more.",
		})
		session.Save(r, w)
		http.Redirect(w, r, basePath+"/new", http.StatusFound)
		return
	}

	formData := ParseCreateForm(r)
	validatedForm := ValidateCreateForm(ValidateCreateFormParams{
		queries:          lh.queries,
		userID:           userID,
		formData:         formData,
		userSubscription: subscription,
	})

	if !validatedForm.IsValid {
		session.AddFlash(validatedForm.Errors)
		session.Save(r, w)
		http.Redirect(w, r, basePath+"/new", http.StatusFound)
		return
	}

	_, err := SaveNewLink(lh.queries, userID, formData)
	if err != nil {
		log.Printf("Failed to create new link: %v", err)
		http.Error(w, "Failed to create new link", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, basePath, http.StatusSeeOther)
}

func (lh *LinkHandler) UserLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session, _ := lh.sessionStore.Get(r, "session")
	user := session.Values["user"].(auth.UserSession)
	userID := user.UserID

	link, err := lh.queries.GetLinkForUser(context.Background(), db.GetLinkForUserParams{
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

	if err := lh.template.ExecuteTemplate(w, "link.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}
