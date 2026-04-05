// helpers.go — parent helper signup routes for PocketBase 0.36.x

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

// ── Role definitions ──────────────────────────────────────────────────────────

type Role struct {
	Key   string
	Label string
}

var AllRoles = []Role{
	{"friday_photo_lit", "Friday Setup - Setting up Photography & Literature Entries (5:30pm - 7:00pm)"},
	{"friday_tables", "Friday Setup - Setting up Tables (5:30pm - 7:00pm)"},
	{"pet_hall", "Show Assist - Pet Hall Setup and Coordination (8:30am - 2:00pm)"},
	{"inflatable", "Entertainment - Inflatable Obstacle Course Supervision (9:00am - 1:30pm)"},
	{"crafts", "Entertainment - Crafts Relief/Support (10:00am - 12:30pm)"},
	{"hot_drinks", "Show Assist - Hot Drinks (10:00am - 12:30pm)"},
	{"judge_refreshments", "Show Assist - Refreshments for Judges (10:15am - 11:45am)"},
	{"wheels_parade", "Wheels Parade - Assistant (11:15am - 12:00pm)"},
	{"sausage_sizzle", "Lunchtime - Sausage Sizzle Assistant (11:30am - 1:30pm)"},
	{"cleanup_bathrooms", "Clean Up - Cleaning Bathrooms (3:30pm)"},
	{"cleanup_kitchen", "Clean Up - Cleaning Kitchen (3:30pm)"},
	{"cleanup_tables", "Clean Up - Putting away Tables and Chairs (3:30pm)"},
	{"cleanup_sweep", "Clean Up - Sweeping the Hall (3:30pm)"},
	{"cleanup_vacuum", "Clean Up - Vacuuming the Foyer & Pet Hall (3:30pm)"},
}

// ── View data types ───────────────────────────────────────────────────────────

type ExistingHelperForm struct {
	ParentName        string
	CellNumber        string
	Email             string
	IsCommitteeMember bool
	HasAllocatedRole  bool
	UnableToVolunteer bool
	MheMembership     string
	selectedRoles     map[string]bool
	HelpersJSON       string
}

// HasRole is called from the template: {{if $.ExistingForm.HasRole .Key}}
func (f *ExistingHelperForm) HasRole(key string) bool {
	return f.selectedRoles[key]
}

// ── Route registration ────────────────────────────────────────────────────────

// ShowHelperForm displays the parent helper signup form
func ShowHelperForm(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := map[string]any{
			"UserEmail":    e.Auth.Email(),
			"Roles":        AllRoles,
			"ExistingForm": nil,
		}

		existing, err := app.FindFirstRecordByData("parent_help", "user", e.Auth.Id)
		if err == nil && existing != nil {
			var roleKeys []string
			rolesRaw, _ := json.Marshal(existing.Get("roles"))
			_ = json.Unmarshal(rolesRaw, &roleKeys)
			roleSet := make(map[string]bool, len(roleKeys))
			for _, k := range roleKeys {
				roleSet[k] = true
			}
			helpersRaw, _ := json.Marshal(existing.Get("helpers"))

			data["ExistingForm"] = &ExistingHelperForm{
				ParentName:        existing.GetString("parent_name"),
				CellNumber:        existing.GetString("cell_number"),
				Email:             existing.GetString("email"),
				IsCommitteeMember: existing.GetBool("is_committee_member"),
				HasAllocatedRole:  existing.GetBool("has_allocated_role"),
				UnableToVolunteer: existing.GetBool("unable_to_volunteer"),
				MheMembership:     existing.GetString("mhe_membership"),
				selectedRoles:     roleSet,
				HelpersJSON:       string(helpersRaw),
			}
		}

		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/helper_signup.html",
		).Render(data)
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

// HandleHelperSubmit processes the helper signup form submission
func HandleHelperSubmit(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return e.BadRequestError("Invalid form data", err)
		}
		form := e.Request.Form

		roles := form["roles"]
		helpers := parseHelpers(form)

		rolesJSON, _ := json.Marshal(roles)
		helpersJSON, _ := json.Marshal(helpers)

		collection, err := app.FindCollectionByNameOrId("parent_help")
		if err != nil {
			return e.InternalServerError("", err)
		}

		record, err := app.FindFirstRecordByData("parent_help", "user", e.Auth.Id)
		if err != nil {
			// No existing record — create new
			record = core.NewRecord(collection)
			record.Set("user", e.Auth.Id)
		}

		record.Set("parent_name", form.Get("parent_name"))
		record.Set("cell_number", form.Get("cell_number"))
		record.Set("email", form.Get("email"))
		record.Set("is_committee_member", form.Get("is_committee_member") == "true")
		record.Set("has_allocated_role", form.Get("has_allocated_role") == "true")
		record.Set("unable_to_volunteer", form.Get("unable_to_volunteer") == "true")
		record.Set("mhe_membership", form.Get("mhe_membership"))
		record.Set("roles", json.RawMessage(rolesJSON))
		record.Set("helpers", json.RawMessage(helpersJSON))

		if err := app.Save(record); err != nil {
			return e.InternalServerError("", err)
		}

		return e.Redirect(http.StatusSeeOther, "/")
	}
}

func registerHelperRoutes(app *pocketbase.PocketBase, registry *template.Registry) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// GET /helper-signup
		se.Router.GET("/helper-signup", func(e *core.RequestEvent) error {
			data := map[string]any{
				"UserEmail":    e.Auth.Email(),
				"Roles":        AllRoles,
				"ExistingForm": nil,
			}

			existing, err := app.FindFirstRecordByData("parent_help", "user", e.Auth.Id)
			if err == nil && existing != nil {
				var roleKeys []string
				rolesRaw, _ := json.Marshal(existing.Get("roles"))
				_ = json.Unmarshal(rolesRaw, &roleKeys)
				roleSet := make(map[string]bool, len(roleKeys))
				for _, k := range roleKeys {
					roleSet[k] = true
				}
				helpersRaw, _ := json.Marshal(existing.Get("helpers"))

				data["ExistingForm"] = &ExistingHelperForm{
					ParentName:        existing.GetString("parent_name"),
					CellNumber:        existing.GetString("cell_number"),
					Email:             existing.GetString("email"),
					IsCommitteeMember: existing.GetBool("is_committee_member"),
					HasAllocatedRole:  existing.GetBool("has_allocated_role"),
					UnableToVolunteer: existing.GetBool("unable_to_volunteer"),
					MheMembership:     existing.GetString("mhe_membership"),
					selectedRoles:     roleSet,
					HelpersJSON:       string(helpersRaw),
				}
			}

			html, err := registry.LoadFiles(
				"views/layout.html",
				"views/helper_signup.html",
			).Render(data)
			if err != nil {
				return e.InternalServerError("", err)
			}
			return e.HTML(http.StatusOK, html)
		}).Bind(apis.RequireAuth())

		// POST /helper-signup
		se.Router.POST("/helper-signup", func(e *core.RequestEvent) error {
			if err := e.Request.ParseForm(); err != nil {
				return e.BadRequestError("Invalid form data", err)
			}
			form := e.Request.Form

			roles := form["roles"]
			helpers := parseHelpers(form)

			rolesJSON, _ := json.Marshal(roles)
			helpersJSON, _ := json.Marshal(helpers)

			collection, err := app.FindCollectionByNameOrId("parent_help")
			if err != nil {
				return e.InternalServerError("", err)
			}

			record, err := app.FindFirstRecordByData("parent_help", "user", e.Auth.Id)
			if err != nil {
				// No existing record — create new
				record = core.NewRecord(collection)
				record.Set("user", e.Auth.Id)
			}

			record.Set("parent_name", form.Get("parent_name"))
			record.Set("cell_number", form.Get("cell_number"))
			record.Set("email", form.Get("email"))
			record.Set("is_committee_member", form.Get("is_committee_member") == "true")
			record.Set("has_allocated_role", form.Get("has_allocated_role") == "true")
			record.Set("unable_to_volunteer", form.Get("unable_to_volunteer") == "true")
			record.Set("mhe_membership", form.Get("mhe_membership"))
			record.Set("roles", json.RawMessage(rolesJSON))
			record.Set("helpers", json.RawMessage(helpersJSON))

			if err := app.Save(record); err != nil {
				return e.InternalServerError("", err)
			}

			return e.Redirect(http.StatusSeeOther, "/")
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseHelpers(form map[string][]string) []map[string]string {
	var helpers []map[string]string
	for i := 0; ; i++ {
		firstName := formGet(form, fmt.Sprintf("helpers[%d][first_name]", i))
		lastName := formGet(form, fmt.Sprintf("helpers[%d][last_name]", i))
		cell := formGet(form, fmt.Sprintf("helpers[%d][cell]", i))
		relationship := formGet(form, fmt.Sprintf("helpers[%d][relationship]", i))
		notes := formGet(form, fmt.Sprintf("helpers[%d][notes]", i))
		if firstName == "" && lastName == "" && cell == "" {
			break
		}
		helpers = append(helpers, map[string]string{
			"first_name":   firstName,
			"last_name":    lastName,
			"cell":         cell,
			"relationship": relationship,
			"notes":        notes,
		})
	}
	return helpers
}

func formGet(form map[string][]string, key string) string {
	if vals, ok := form[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// HasHelperSignup checks if a user has a completed helper signup record.
func HasHelperSignup(app *pocketbase.PocketBase, userID string) bool {
	record, err := app.FindFirstRecordByData("parent_help", "user", userID)
	return err == nil && record != nil
}
