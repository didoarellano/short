package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

type SessionStore interface {
	Get(r *http.Request, name string) (*sessions.Session, error)
	Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error
}
