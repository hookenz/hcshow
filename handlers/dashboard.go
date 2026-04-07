package handlers

import (
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

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

		// Calculate registration completion flags
		hasChildren := false
		hasExhibits := false
		hasHelperSignup := false

		// Check if user has any children (exhibitors)
		childRecords, err := app.FindRecordsByFilter("exhibitor", "user = {:userId}", "", 100, 0, dbx.Params{"userId": e.Auth.Id})
		if err == nil && len(childRecords) > 0 {
			hasChildren = true

			// Check if ALL children have at least one exhibit entry
			allChildrenHaveExhibits := true
			for _, child := range childRecords {
				exhibits, err := app.FindRecordsByFilter("exhibits", "exhibitor = {:childId}", "", 1, 0, dbx.Params{"childId": child.Id})
				if err != nil || len(exhibits) == 0 {
					allChildrenHaveExhibits = false
					break
				}
			}
			hasExhibits = allChildrenHaveExhibits
		}

		// Check if user has helper signup record
		hasHelperSignup = HasHelperSignup(app, e.Auth.Id)

		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/dashboard.html",
		).Render(map[string]any{
			"UserEmail":       e.Auth.Email(),
			"ShowDate":        showDate,
			"ShowDateISO":     showDateISO,
			"HasChildren":     hasChildren,
			"HasExhibits":     hasExhibits,
			"HasHelperSignup": hasHelperSignup,
		})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}
