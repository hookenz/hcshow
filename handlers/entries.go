package handlers

import (
	"cmp"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

type CategoryOption struct {
	CategoryID     string
	CategoryName   string
	SectionName    string
	ShortCode      string
	AlreadyEntered bool
}

type EntriesPageData struct {
	ExhibitorID  string
	FirstName    string
	LastName     string
	AgeOnShowDay int
	AgeGroupName string
	AgeGroupID   string
	Categories   []CategoryOption
	NoAgeGroup   bool
	Error        string
}

type EntryRow struct {
	EntryID      string
	ExhibitorID  string
	CategoryName string
	SectionName  string
	ShortCode    string
	Status       string
}

func ShowEntries(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data, err := buildEntriesData(app, e)
		if err != nil {
			return e.InternalServerError("", err)
		}

		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/entries.html",
		).Render(data)
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func CreateEntry(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		exhibitorID := e.Request.PathValue("id")

		exhibitor, err := app.FindRecordById("exhibitor", exhibitorID)
		if err != nil {
			return e.NotFoundError("", err)
		}
		if exhibitor.GetString("user") != e.Auth.Id {
			return e.ForbiddenError("", nil)
		}

		formData := struct {
			CategoryID string `form:"category_id"`
			AgeGroupID string `form:"age_group_id"`
		}{}
		if err := e.BindBody(&formData); err != nil {
			return e.BadRequestError("", err)
		}

		// Add this temporarily
		app.Logger().Info("create entry",
			"exhibitorID", exhibitorID,
			"categoryID", formData.CategoryID,
			"ageGroupID", formData.AgeGroupID,
		)

		if formData.CategoryID == "" || formData.AgeGroupID == "" {
			return e.BadRequestError("Missing category or age group", nil)
		}

		collection, err := app.FindCollectionByNameOrId("exhibits")
		if err != nil {
			return e.InternalServerError("", err)
		}

		record := core.NewRecord(collection)
		record.Set("exhibitor", exhibitorID)
		record.Set("category", formData.CategoryID)
		record.Set("age_group", formData.AgeGroupID)
		record.Set("status", "pending")

		if err := app.Save(record); err != nil {
			app.Logger().Error("save failed", "err", err.Error())
			return e.InternalServerError("save failed: "+err.Error(), err)
		}

		// Fetch updated entries list and return as fragment
		entries, err := fetchEntryRows(app, exhibitorID)
		if err != nil {
			return e.InternalServerError("", err)
		}

		html, err := registry.LoadFiles(
			"views/partials/entries_list.html",
		).Render(map[string]any{
			"Entries": entries,
		})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func ListEntries(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		exhibitorID := e.Request.PathValue("id")

		exhibitor, err := app.FindRecordById("exhibitor", exhibitorID)
		if err != nil {
			return e.NotFoundError("", err)
		}
		if exhibitor.GetString("user") != e.Auth.Id {
			return e.ForbiddenError("", nil)
		}

		rows, err := fetchEntryRows(app, exhibitorID)
		if err != nil {
			return e.InternalServerError("", err)
		}

		html, err := registry.LoadFiles(
			"views/partials/entries_list.html",
		).Render(map[string]any{
			"Entries": rows,
		})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func DeleteEntry(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		exhibitorID := e.Request.PathValue("id")

		exhibitor, err := app.FindRecordById("exhibitor", exhibitorID)
		if err != nil {
			return e.NotFoundError("", err)
		}
		if exhibitor.GetString("user") != e.Auth.Id {
			return e.ForbiddenError("", nil)
		}

		entry, err := app.FindRecordById("exhibits", e.Request.PathValue("entryid"))
		if err != nil {
			return e.NotFoundError("", err)
		}
		if entry.GetString("exhibitor") != exhibitorID {
			return e.ForbiddenError("", nil)
		}

		if err := app.Delete(entry); err != nil {
			return e.InternalServerError("", err)
		}

		rows, err := fetchEntryRows(app, exhibitorID)
		if err != nil {
			return e.InternalServerError("", err)
		}

		html, err := registry.LoadFiles(
			"views/partials/entries_list.html",
		).Render(map[string]any{
			"Entries": rows,
		})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

// buildEntriesData fetches and assembles everything needed for the entries page
func buildEntriesData(app *pocketbase.PocketBase, e *core.RequestEvent) (*EntriesPageData, error) {
	exhibitorID := e.Request.PathValue("id")

	exhibitor, err := app.FindRecordById("exhibitor", exhibitorID)
	if err != nil {
		return nil, err
	}
	if exhibitor.GetString("user") != e.Auth.Id {
		return nil, fmt.Errorf("forbidden")
	}

	data := &EntriesPageData{
		ExhibitorID: exhibitorID,
		FirstName:   exhibitor.GetString("first_name"),
		LastName:    exhibitor.GetString("last_name"),
	}

	// Get show date
	settings, err := app.FindFirstRecordByFilter(
		"settings",
		"show_date != ''",
		dbx.Params{},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			data.NoAgeGroup = true
			return data, nil
		}
		return nil, err
	}
	showDate := settings.GetDateTime("show_date").Time()

	// Calculate age on show date
	birthDate := exhibitor.GetDateTime("birth_date").Time()
	if birthDate.IsZero() {
		data.NoAgeGroup = true
		return data, nil
	}

	ageOnShowDay := showDate.Year() - birthDate.Year()
	if showDate.Month() < birthDate.Month() ||
		(showDate.Month() == birthDate.Month() && showDate.Day() < birthDate.Day()) {
		ageOnShowDay--
	}
	data.AgeOnShowDay = ageOnShowDay

	// Find matching age group directly
	ageGroup, err := app.FindFirstRecordByFilter(
		"age_group",
		"min <= {:age} && max >= {:age}",
		dbx.Params{"age": ageOnShowDay},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			data.NoAgeGroup = true
			return data, nil
		}
		return nil, err
	}

	data.AgeGroupID = ageGroup.Id
	data.AgeGroupName = ageGroup.GetString("name")

	// Fetch eligible category_age_group records for this age group
	cagRecords, err := app.FindRecordsByFilter(
		"category_age_group",
		"age_group = {:ag}",
		"",
		500, 0,
		dbx.Params{"ag": ageGroup.Id},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Expand category on cag records
	app.ExpandRecords(cagRecords, []string{"category"}, nil)

	// Collect category records and expand section on them
	var categoryRecords []*core.Record
	for _, cag := range cagRecords {
		if cat := cag.ExpandedOne("category"); cat != nil {
			categoryRecords = append(categoryRecords, cat)
		}
	}
	app.ExpandRecords(categoryRecords, []string{"section"}, nil)

	// Fetch existing entries for this exhibitor
	existing, err := app.FindRecordsByFilter(
		"exhibits",
		"exhibitor = {:ex}",
		"", 500, 0,
		dbx.Params{"ex": exhibitorID},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	enteredCategories := map[string]bool{}
	for _, ex := range existing {
		enteredCategories[ex.GetString("category")] = true
	}

	// Build category options, active only
	for _, cag := range cagRecords {
		cat := cag.ExpandedOne("category")
		if cat == nil || !cat.GetBool("active") {
			continue
		}
		sectionName := ""
		if section := cat.ExpandedOne("section"); section != nil {
			sectionName = section.GetString("name")
		}
		data.Categories = append(data.Categories, CategoryOption{
			CategoryID:     cat.Id,
			CategoryName:   cat.GetString("name"),
			SectionName:    sectionName,
			ShortCode:      cat.GetString("short_code"),
			AlreadyEntered: enteredCategories[cat.Id],
		})
	}

	slices.SortFunc(data.Categories, func(a, b CategoryOption) int {
		if n := cmp.Compare(a.SectionName, b.SectionName); n != 0 {
			return n
		}
		return cmp.Compare(a.CategoryName, b.CategoryName)
	})

	return data, nil
}

func fetchEntryRows(app *pocketbase.PocketBase, exhibitorID string) ([]EntryRow, error) {
	entries, err := app.FindRecordsByFilter(
		"exhibits",
		"exhibitor = {:ex}",
		"category",
		500, 0,
		dbx.Params{"ex": exhibitorID},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	app.ExpandRecords(entries, []string{"category"}, nil)

	var categoryRecords []*core.Record
	for _, entry := range entries {
		if cat := entry.ExpandedOne("category"); cat != nil {
			categoryRecords = append(categoryRecords, cat)
		}
	}
	app.ExpandRecords(categoryRecords, []string{"section"}, nil)

	rows := []EntryRow{}
	for _, entry := range entries {
		cat := entry.ExpandedOne("category")
		if cat == nil {
			continue
		}
		sectionName := ""
		if section := cat.ExpandedOne("section"); section != nil {
			sectionName = section.GetString("name")
		}
		rows = append(rows, EntryRow{
			EntryID:      entry.Id,
			ExhibitorID:  exhibitorID,
			CategoryName: cat.GetString("name"),
			SectionName:  sectionName,
			ShortCode:    cat.GetString("short_code"),
			Status:       entry.GetString("status"),
		})
	}
	return rows, nil
}
