package links

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/didoarellano/short/internal/auth"
	"github.com/didoarellano/short/internal/config"
	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/shortcode"
	"github.com/didoarellano/short/internal/templ"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type FormData struct {
	DestinationUrl  string
	Path            string
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
	Message        string
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

type FormValidation struct {
	IsValid bool
	Errors  FormValidationErrors
}

type ShowCreateFormParams struct {
	w                http.ResponseWriter
	r                *http.Request
	session          *sessions.Session
	template         *templ.Templ
	userSubscription auth.Subscription
}

func ShowCreateForm(arg ShowCreateFormParams) {
	var validationErrors FormValidationErrors
	flashes := arg.session.Flashes()
	if len(flashes) > 0 {
		if v, ok := flashes[0].(FormValidationErrors); ok {
			validationErrors = v
		}
	}
	data := map[string]interface{}{
		"validationErrors": validationErrors,
		"userSubscription": arg.userSubscription,
	}
	arg.session.Save(arg.r, arg.w)
	if err := arg.template.ExecuteTemplate(arg.w, "create_link.html", data); err != nil {
		http.Error(arg.w, "Failed to render template", http.StatusInternalServerError)
	}
}

func ParseCreateForm(r *http.Request) FormData {
	r.ParseForm()
	formData := FormData{
		DestinationUrl:  strings.TrimSpace(r.FormValue("url")),
		Path:            strings.TrimSpace(r.FormValue("path")),
		Title:           strings.TrimSpace(r.FormValue("title")),
		Notes:           strings.TrimSpace(r.FormValue("notes")),
		CreateDuplicate: r.FormValue("create-duplicate") == "on",
	}
	return formData
}

type ValidateCreateFormParams struct {
	queries          *db.Queries
	userID           int32
	formData         FormData
	userSubscription auth.Subscription
}

func ValidateCreateForm(arg ValidateCreateFormParams) FormValidation {
	formData := arg.formData

	validation := FormValidation{
		IsValid: true,
		Errors: FormValidationErrors{
			FormFields: map[string]FormFieldValidation{
				"Url": {
					Value: formData.DestinationUrl,
				},
				"Path": {
					Value: formData.Path,
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
		},
	}

	if formData.DestinationUrl == "" {
		validation.IsValid = false
		validation.Errors.FormFields["Url"] = FormFieldValidation{
			Value:   formData.DestinationUrl,
			Message: "Destination URL is required",
		}
		return validation
	}

	if formData.CreateDuplicate && !arg.userSubscription.CanCreateDuplicates {
		validation.IsValid = false
	}

	if formData.Path != "" && !arg.userSubscription.CanCustomisePath {
		validation.IsValid = false
		validation.Errors.FormFields["Path"] = FormFieldValidation{
			Value:   formData.Path,
			Message: "Custom paths require a pro subscription",
		}
	}

	if formData.Path != "" && arg.userSubscription.CanCustomisePath {
		customPathConfig, _ := config.LoadCustomPathConfig()
		err := ValidateCustomPath(formData.Path, customPathConfig)

		if err != nil {
			validation.IsValid = false
			validation.Errors.FormFields["Path"] = FormFieldValidation{
				Value:   formData.Path,
				Message: err.Error(),
			}
		} else {
			link, err := arg.queries.GetLinkByShortCode(context.Background(), formData.Path)
			if err != pgx.ErrNoRows {
				validation.IsValid = false
				validation.Errors.FormFields["Path"] = FormFieldValidation{
					Value:   formData.Path,
					Message: fmt.Sprintf("%s is already in use", formData.Path),
				}

				if link.UserID == arg.userID {
					validation.Errors.Duplicates = DuplicateUrls{
						Urls: []DuplicateUrl{{
							Text: link.ShortCode,
							Href: fmt.Sprintf("/%s/links/%s", config.AppData.AppPathPrefix, link.ShortCode),
						}},
						Message: "You've used this path before",
					}
				}
			}
		}
	}

	if !formData.CreateDuplicate {
		links, _ := arg.queries.FindDuplicatesForUrl(context.Background(), db.FindDuplicatesForUrlParams{
			UserID:         arg.userID,
			DestinationUrl: formData.DestinationUrl,
			Limit:          3,
		})

		if len(links.ShortCodes) > 0 {
			duplicates := findDuplicateLinks(arg.queries, arg.userID, formData.DestinationUrl)
			if duplicates != nil {
				validation.IsValid = false
				validation.Errors.Duplicates = *duplicates
				validation.Errors.Duplicates.Message = "You've shortened this link before"
			}
		}
	}

	return validation
}

func findDuplicateLinks(queries *db.Queries, userID int32, destinationUrl string) *DuplicateUrls {
	links, _ := queries.FindDuplicatesForUrl(context.Background(), db.FindDuplicatesForUrlParams{
		UserID:         userID,
		DestinationUrl: destinationUrl,
		Limit:          3,
	})

	if len(links.ShortCodes) == 0 {
		return nil
	}

	duplicates := &DuplicateUrls{
		DestinationUrl: destinationUrl,
		RemainingCount: links.RemainingCount,
	}

	for _, shortcode := range links.ShortCodes {
		duplicates.Urls = append(duplicates.Urls, DuplicateUrl{
			Href: fmt.Sprintf("/%s/links/%s", config.AppData.AppPathPrefix, shortcode),
			Text: shortcode,
		})
	}

	return duplicates
}

func SaveNewLink(queries *db.Queries, userID int32, formData FormData) (db.Link, error) {
	var shortCode string
	if formData.Path != "" {
		shortCode = formData.Path
	} else {
		shortCode = shortcode.New(userID, formData.DestinationUrl, 7)
	}

	return queries.CreateLink(context.Background(), db.CreateLinkParams{
		UserID:         userID,
		ShortCode:      shortCode,
		DestinationUrl: formData.DestinationUrl,
		Title:          pgtype.Text{String: formData.Title, Valid: true},
		Notes:          pgtype.Text{String: formData.Notes, Valid: true},
	})
}

func ValidateCustomPath(path string, config *config.CustomPathConfig) error {
	length := len(path)
	if length < config.MinLength || length > config.MaxLength {
		return fmt.Errorf("path must be between %d and %d characters", config.MinLength, config.MaxLength)
	}

	for _, word := range config.ReservedWords {
		if strings.EqualFold(path, word) {
			return errors.New("path is reserved")
		}
	}

	return nil
}
