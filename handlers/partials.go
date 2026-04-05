package handlers

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"

	"hcshow/models"
)

func ListExhibitors(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return renderExhibitorList(app, registry, e)
	}
}

// renderExhibitorList is shared by ListExhibitors and DeleteExhibitor
func renderExhibitorList(app *pocketbase.PocketBase, registry *template.Registry, e *core.RequestEvent) error {
	records, err := app.FindRecordsByFilter(
		"exhibitor",
		"user = {:user}",
		"first_name,last_name",
		100, 0,
		dbx.Params{"user": e.Auth.Id},
	)
	if err != nil {
		return e.InternalServerError("", err)
	}

	rows := make([]models.ExhibitorRow, len(records))
	for i, r := range records {
		row := models.BuildExhibitorRow(
			r.Id,
			r.GetString("exhibitor_id"),
			r.GetString("first_name"),
			r.GetString("last_name"),
			r.GetDateTime("birth_date"),
		)

		// Count entries for this exhibitor
		entries, err := app.FindRecordsByFilter("exhibits", "exhibitor = {:id}", "", 100, 0, dbx.Params{"id": r.Id})
		if err == nil {
			row.EntryCount = len(entries)
		}

		// Find age group for this exhibitor (convert age to float64 for comparison)
		ageGroups, err := app.FindRecordsByFilter("age_group", "min <= {:age} && max >= {:age}", "", 100, 0, dbx.Params{"age": float64(row.Age)})
		if err == nil && len(ageGroups) > 0 {
			row.AgeGroupName = ageGroups[0].GetString("name")
		}

		rows[i] = row
	}

	html, err := registry.LoadFiles(
		"views/partials/exhibitors.html",
	).Render(map[string]any{
		"Exhibitors": rows,
	})
	if err != nil {
		return e.InternalServerError("", err)
	}
	return e.HTML(http.StatusOK, html)
}
