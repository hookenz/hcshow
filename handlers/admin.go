package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
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
		settings, err := app.FindFirstRecordByFilter("settings", "", nil)
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

		records, err := app.FindRecordsByFilter("entries_by_age_group", "", "", 0, 0)
		if err != nil {
			return e.InternalServerError("", err)
		}

		type Row struct {
			Name  string
			Count int
		}

		rows := []Row{}
		total := 0

		for _, rec := range records {
			c := rec.GetInt("count")
			rows = append(rows, Row{
				Name:  rec.GetString("name"),
				Count: c,
			})
			total += c
		}

		m := maroto.New()

		// Title
		m.AddRow(20,
			text.NewCol(12,
				"Exhibitor Age Group Summary",
				props.Text{
					Top:   6,
					Size:  18,
					Align: align.Center,
					Style: fontstyle.Bold,
				},
			),
		)

		// Generated date
		m.AddRow(10,
			text.NewCol(12,
				fmt.Sprintf("Generated: %s", time.Now().Format("2 Jan 2006 15:04")),
				props.Text{
					Align: align.Right,
					Size:  9,
				},
			),
		)

		// Spacer
		m.AddRow(5)

		// Header
		m.AddRow(10,
			text.NewCol(8,
				"Age Group",
				props.Text{
					Style: fontstyle.Bold,
					Align: align.Left,
				},
			),
			text.NewCol(4,
				"Exhibitors",
				props.Text{
					Style: fontstyle.Bold,
					Align: align.Right,
				},
			),
		)

		m.AddRow(1,
			line.NewCol(12),
		)

		// Data rows
		for _, r := range rows {
			m.AddRow(8,
				text.NewCol(8,
					r.Name,
					props.Text{
						Align: align.Left,
					},
				),
				text.NewCol(4,
					fmt.Sprintf("%d", r.Count),
					props.Text{
						Align: align.Right,
					},
				),
			)
		}

		// Divider before total
		m.AddRow(2,
			line.NewCol(12),
		)

		// Total row
		m.AddRow(10,
			text.NewCol(8,
				"TOTAL",
				props.Text{
					Style: fontstyle.Bold,
					Align: align.Left,
				},
			),
			text.NewCol(4,
				fmt.Sprintf("%d", total),
				props.Text{
					Style: fontstyle.Bold,
					Align: align.Right,
				},
			),
		)

		doc, err := m.Generate()
		if err != nil {
			return e.InternalServerError("", err)
		}

		pdfBytes := doc.GetBytes()

		e.Response.Header().Set("Content-Type", "application/pdf")
		e.Response.Header().Set("Content-Disposition", "inline; filename=age-group-summary.pdf")

		return e.Blob(http.StatusOK, "application/pdf", pdfBytes)
	}
}

func PrintHallPlanningReport(app *pocketbase.PocketBase, registry *template.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {

		records, err := app.FindRecordsByFilter("entries_by_category_age_group", "", "", 0, 0)
		if err != nil {
			return e.InternalServerError("", err)
		}

		// --- Structures ---
		type AgeGroup struct {
			Name  string
			Count int
		}

		type Category struct {
			Name      string
			AgeGroups []AgeGroup
			Total     int
		}

		type Section struct {
			Name       string
			Categories map[string]*Category
			Total      int
		}

		sections := map[string]*Section{}
		grandTotal := 0

		// --- Build hierarchy ---
		for _, rec := range records {

			sectionName := rec.GetString("section")
			categoryName := rec.GetString("category")
			ageGroupName := rec.GetString("age_group")
			count := rec.GetInt("count")

			// Section
			if _, ok := sections[sectionName]; !ok {
				sections[sectionName] = &Section{
					Name:       sectionName,
					Categories: map[string]*Category{},
				}
			}
			sec := sections[sectionName]

			// Category
			if _, ok := sec.Categories[categoryName]; !ok {
				sec.Categories[categoryName] = &Category{
					Name: categoryName,
				}
			}
			cat := sec.Categories[categoryName]

			// Age group row
			cat.AgeGroups = append(cat.AgeGroups, AgeGroup{
				Name:  ageGroupName,
				Count: count,
			})

			// Totals
			cat.Total += count
			sec.Total += count
			grandTotal += count
		}

		// --- PDF ---
		m := maroto.New()

		// Title
		m.AddRow(15,
			text.NewCol(12, "Hall Planning Report",
				props.Text{
					Align: align.Center,
					Style: fontstyle.Bold,
					Size:  16,
				}),
		)

		// Generated timestamp
		m.AddRow(8,
			text.NewCol(12,
				fmt.Sprintf("Generated: %s", time.Now().Format("2 Jan 2006 15:04")),
				props.Text{Align: align.Right, Size: 9},
			),
		)

		m.AddRow(5)

		// --- Render hierarchy ---
		for _, sec := range sections {

			// Section header
			m.AddRow(10,
				text.NewCol(12, sec.Name,
					props.Text{Style: fontstyle.Bold, Size: 13}),
			)

			for _, cat := range sec.Categories {

				// Category header
				m.AddRow(8,
					text.NewCol(12, "  "+cat.Name,
						props.Text{Style: fontstyle.Bold}),
				)

				// Column headers
				// m.AddRow(8,
				// 	text.NewCol(8, "    Age Group", props.Text{Style: fontstyle.Bold}),
				// 	text.NewCol(4, "Count", props.Text{
				// 		Style: fontstyle.Bold,
				// 		Align: align.Right,
				// 	}),
				// )

				m.AddRow(1, line.NewCol(12))

				// Age group rows
				for _, ag := range cat.AgeGroups {
					m.AddRow(7,
						text.NewCol(8, "    "+ag.Name),
						text.NewCol(4,
							fmt.Sprintf("%d", ag.Count),
							props.Text{Align: align.Right},
						),
					)
				}

				// Category total
				m.AddRow(7,
					text.NewCol(8, "    Total"),
					text.NewCol(4,
						fmt.Sprintf("%d", cat.Total),
						props.Text{
							Align: align.Right,
							Style: fontstyle.Bold,
						},
					),
				)

				m.AddRow(4)
			}

			// Section total
			m.AddRow(8,
				text.NewCol(8, "Section Total"),
				text.NewCol(4,
					fmt.Sprintf("%d", sec.Total),
					props.Text{
						Align: align.Right,
						Style: fontstyle.Bold,
					},
				),
			)

			// m.AddRow(5, line.NewCol(12))
			m.AddRow(5)
		}

		// Grand total
		m.AddRow(10,
			text.NewCol(8, "Total Entries",
				props.Text{Style: fontstyle.Bold}),
			text.NewCol(4,
				fmt.Sprintf("%d", grandTotal),
				props.Text{
					Align: align.Right,
					Style: fontstyle.Bold,
				},
			),
		)

		doc, err := m.Generate()
		if err != nil {
			return e.InternalServerError("", err)
		}

		return e.Blob(http.StatusOK, "application/pdf", doc.GetBytes())
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
