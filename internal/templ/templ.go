package templ

import (
	"html/template"
	"net/http"

	"github.com/didoarellano/short/internal/config"
	"github.com/didoarellano/short/internal/session"
)

type Templ struct {
	AppData      *config.GlobalAppData
	sessionStore session.SessionStore
	t            *template.Template
}

func New(t *template.Template, AppData *config.GlobalAppData, s session.SessionStore) *Templ {
	return &Templ{
		AppData:      AppData,
		t:            t,
		sessionStore: s,
	}
}

func (templ *Templ) ExecuteTemplate(w http.ResponseWriter, templateName string, localData map[string]interface{}) error {
	data := make(map[string]interface{})
	for k, v := range localData {
		data[k] = v
	}
	data["AppPathPrefix"] = templ.AppData.AppPathPrefix
	data["RedirectorBaseURL"] = templ.AppData.RedirectorBaseURL
	return templ.t.ExecuteTemplate(w, templateName, data)
}

func (templ *Templ) RenderStatic(templateName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := templ.sessionStore.Get(r, "session")
		user := session.Values["user"]
		data := map[string]interface{}{
			"user": user,
		}
		if err := templ.ExecuteTemplate(w, templateName, data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
