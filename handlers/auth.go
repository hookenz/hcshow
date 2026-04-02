package handlers

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

func ShowLogin(registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/login.html",
		).Render(map[string]any{"Error": ""})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func HandleLogin(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := struct {
			Email    string `form:"email"`
			Password string `form:"password"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return e.BadRequestError("Invalid form data", err)
		}

		record, err := app.FindAuthRecordByEmail("users", data.Email)
		if err != nil || !record.ValidatePassword(data.Password) {
			html, err := registry.LoadFiles(
				"views/layout.html",
				"views/login.html",
			).Render(map[string]any{"Error": "Invalid email or password"})
			if err != nil {
				return e.InternalServerError("", err)
			}
			return e.HTML(http.StatusUnauthorized, html)
		}

		token, err := record.NewAuthToken()
		if err != nil {
			return e.InternalServerError("Could not create token", err)
		}

		http.SetCookie(e.Response, &http.Cookie{
			Name:     "pb_auth",
			Value:    token,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})

		e.Response.Header().Set("HX-Redirect", "/")
		return e.NoContent(http.StatusOK)
	}
}

func ShowRegister(registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/register.html",
		).Render(map[string]any{"Error": ""})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func HandleRegister(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := struct {
			Name            string `form:"name"`
			Email           string `form:"email"`
			Password        string `form:"password"`
			PasswordConfirm string `form:"password_confirm"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return e.BadRequestError("Invalid form data", err)
		}

		renderError := func(msg string) error {
			html, err := registry.LoadFiles(
				"views/layout.html",
				"views/register.html",
			).Render(map[string]any{"Error": msg})
			if err != nil {
				return e.InternalServerError("", err)
			}
			return e.HTML(http.StatusUnprocessableEntity, html)
		}

		if data.Password != data.PasswordConfirm {
			return renderError("Passwords do not match")
		}
		if len(data.Password) < 8 {
			return renderError("Password must be at least 8 characters")
		}

		existing, _ := app.FindAuthRecordByEmail("users", data.Email)
		if existing != nil {
			return renderError("An account with that email already exists")
		}

		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return e.InternalServerError("Could not find users collection", err)
		}

		record := core.NewRecord(collection)
		record.Set("name", data.Name)
		record.Set("email", data.Email)
		record.Set("password", data.Password)
		record.Set("passwordConfirm", data.PasswordConfirm)

		if err := app.Save(record); err != nil {
			return renderError("Could not create account: " + err.Error())
		}

		token, err := record.NewAuthToken()
		if err != nil {
			return e.InternalServerError("Could not create token", err)
		}

		http.SetCookie(e.Response, &http.Cookie{
			Name:     "pb_auth",
			Value:    token,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})

		e.Response.Header().Set("HX-Redirect", "/")
		return e.NoContent(http.StatusOK)
	}
}

func Dashboard(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {

		// Fetch show date from settings
		var showDate string
		var showDateISO string
		settings, err := app.FindFirstRecordByFilter("settings", "show_date != ''", nil)
		if err == nil {
			dt := settings.GetDateTime("show_date")
			if !dt.Time().IsZero() {
				showDate = dt.Time().Format("Monday 2 January 2006")
				showDateISO = dt.Time().Format(time.RFC3339)
			}
		}

		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/dashboard.html",
		).Render(map[string]any{
			"UserEmail":   e.Auth.Email(),
			"ShowDate":    showDate,
			"ShowDateISO": showDateISO,
		})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func Logout() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		http.SetCookie(e.Response, &http.Cookie{
			Name:   "pb_auth",
			Value:  "",
			MaxAge: -1,
			Path:   "/",
		})
		e.Response.Header().Set("HX-Redirect", "/login")
		return e.NoContent(http.StatusOK)
	}
}
