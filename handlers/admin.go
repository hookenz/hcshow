package handlers

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

type AdminDashboardData struct {
	UserEmail               string
	CurrentPhase            string // "before_open", "open", "closed", "prep", "show_day", "after_show"
	RegistrationOpenDate    string
	RegistrationClosingDate string
	ShowDate                string
	NowISO                  string
}

func AdminDashboard(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := AdminDashboardData{
			UserEmail: e.Auth.Email(),
			NowISO:    time.Now().UTC().Format(time.RFC3339),
		}

		// Fetch dates from settings
		settings, err := app.FindFirstRecordByFilter("settings", "1=1", nil)
		if err == nil && settings != nil {
			// Get registration open date
			if openDate := settings.GetDateTime("registration_open_date"); !openDate.Time().IsZero() {
				data.RegistrationOpenDate = openDate.Time().Format(time.RFC3339)
			}

			// Get registration closing date
			if closeDate := settings.GetDateTime("registration_close_date"); !closeDate.Time().IsZero() {
				data.RegistrationClosingDate = closeDate.Time().Format(time.RFC3339)
			}

			// Get show date
			if showDate := settings.GetDateTime("show_date"); !showDate.Time().IsZero() {
				data.ShowDate = showDate.Time().Format("Monday 2 January 2006")
			}
		}

		// Determine current phase
		now := time.Now()
		openDt, _ := time.Parse(time.RFC3339, data.RegistrationOpenDate)
		closeDt, _ := time.Parse(time.RFC3339, data.RegistrationClosingDate)
		showDt, _ := time.Parse(time.RFC3339, data.ShowDate)

		switch {
		case now.Before(openDt):
			data.CurrentPhase = "before_open"
		case now.Before(closeDt):
			data.CurrentPhase = "open"
		case now.Before(showDt):
			data.CurrentPhase = "prep"
		case now.Before(showDt.Add(24 * time.Hour)):
			data.CurrentPhase = "show_day"
		default:
			data.CurrentPhase = "after_show"
		}

		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/admin_dashboard.html",
		).Render(data)
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

// Report stub handlers
func PrintEntryCardsReport(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{"status": "stub"})
	}
}

func PrintExhibitorAgeGroupReport(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{"status": "stub"})
	}
}

func PrintHallPlanningReport(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{"status": "stub"})
	}
}

func PrintPreShowStatsReport(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{"status": "stub"})
	}
}

func CheckTableCards(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{"status": "stub"})
	}
}
