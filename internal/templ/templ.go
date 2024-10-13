package templ

import (
	"html/template"
	"net/http"

	"github.com/didoarellano/short/internal/config"
)

type Templ struct {
	AppData *config.GlobalAppData
	t       *template.Template
}

func New(t *template.Template, AppData *config.GlobalAppData) *Templ {
	return &Templ{
		AppData: AppData,
		t:       t,
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
		if err := templ.ExecuteTemplate(w, templateName, nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
