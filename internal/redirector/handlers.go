package redirector

import (
	"context"
	"log"
	"net/http"

	"github.com/didoarellano/short/internal/db"
	"github.com/gorilla/mux"
)

func RedirectHandler(q *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		shortcode := vars["shortcode"]
		link, err := q.GetDestinationUrl(context.Background(), shortcode)
		if err != nil {
			log.Printf("Destination URL not found: %v", err)
			return
		}
		http.Redirect(w, r, link, http.StatusSeeOther)
	}
}
