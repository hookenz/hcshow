package handlers

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"

	"hcshow/models"
)

func NewExhibitorForm(registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/exhibitor_form.html",
		).Render(map[string]any{
			"Exhibitor": nil,
			"Error":     "",
		})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func CreateExhibitor(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := struct {
			FirstName string `form:"first_name"`
			LastName  string `form:"last_name"`
			BirthDate string `form:"birth_date"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return e.BadRequestError("", err)
		}

		renderError := func(msg string) error {
			html, err := registry.LoadFiles(
				"views/layout.html",
				"views/exhibitor_form.html",
			).Render(map[string]any{"Exhibitor": nil, "Error": msg})
			if err != nil {
				return e.InternalServerError("", err)
			}
			return e.HTML(http.StatusUnprocessableEntity, html)
		}

		collection, err := app.FindCollectionByNameOrId("exhibitor")
		if err != nil {
			return e.InternalServerError("", err)
		}

		var eid string
		for range 5 {
			candidate, err := models.GenerateExhibitorID()
			if err != nil {
				return e.InternalServerError("Could not generate exhibitor ID", err)
			}
			existing, _ := app.FindFirstRecordByData("exhibitor", "exhibitor_id", candidate)
			if existing == nil {
				eid = candidate
				break
			}
		}
		if eid == "" {
			return renderError("Could not generate a unique ID, please try again")
		}

		record := core.NewRecord(collection)
		record.Set("user", e.Auth.Id)
		record.Set("exhibitor_id", eid)
		record.Set("first_name", data.FirstName)
		record.Set("last_name", data.LastName)
		record.Set("birth_date", data.BirthDate)

		if err := app.Save(record); err != nil {
			return renderError("Could not save: " + err.Error())
		}

		http.Redirect(e.Response, e.Request, "/", http.StatusSeeOther)
		return nil
	}
}

func EditExhibitorForm(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		record, err := app.FindRecordById("exhibitor", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("", err)
		}
		if record.GetString("user") != e.Auth.Id {
			return e.ForbiddenError("", nil)
		}

		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/exhibitor_form.html",
		).Render(map[string]any{
			"Exhibitor": record,
			"BirthDate": models.BirthDateForInput(record.GetDateTime("birth_date")),
			"Error":     "",
		})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func UpdateExhibitor(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		record, err := app.FindRecordById("exhibitor", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("", err)
		}
		if record.GetString("user") != e.Auth.Id {
			return e.ForbiddenError("", nil)
		}

		data := struct {
			FirstName string `form:"first_name"`
			LastName  string `form:"last_name"`
			BirthDate string `form:"birth_date"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return e.BadRequestError("", err)
		}

		record.Set("first_name", data.FirstName)
		record.Set("last_name", data.LastName)
		record.Set("birth_date", data.BirthDate)

		if err := app.Save(record); err != nil {
			html, _ := registry.LoadFiles(
				"views/layout.html",
				"views/exhibitor_form.html",
			).Render(map[string]any{
				"Exhibitor": record,
				"BirthDate": models.BirthDateForInput(record.GetDateTime("birth_date")),
				"Error":     "Could not save: " + err.Error(),
			})
			return e.HTML(http.StatusUnprocessableEntity, html)
		}

		http.Redirect(e.Response, e.Request, "/", http.StatusSeeOther)
		return nil
	}
}

func DeleteExhibitor(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		record, err := app.FindRecordById("exhibitor", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("", err)
		}
		if record.GetString("user") != e.Auth.Id {
			return e.ForbiddenError("", nil)
		}
		if err := app.Delete(record); err != nil {
			return e.InternalServerError("", err)
		}

		return renderExhibitorList(app, registry, e)
	}
}
