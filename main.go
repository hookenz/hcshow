package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"

	"hcshow/handlers"
	"hcshow/internal/security"
	"hcshow/middleware"
)

func main() {
	app := pocketbase.New()

	altchakey := security.GetAltchaKey()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registry := template.NewRegistry()

		// Public routes
		se.Router.GET("/login", handlers.ShowLogin(registry))
		se.Router.POST("/login", handlers.HandleLogin(app, registry))
		se.Router.GET("/register", handlers.ShowRegister(registry))
		se.Router.POST("/register", handlers.HandleRegister(app, registry, altchakey))
		se.Router.GET("/api/altcha", handlers.AltchaChallenge(altchakey))

		// Protected routes
		protected := se.Router.Group("")
		protected.BindFunc(middleware.RequireAuth(app))

		protected.GET("/", handlers.Dashboard(app, registry))
		protected.POST("/logout", handlers.Logout())

		protected.GET("/partials/exhibitors", handlers.ListExhibitors(app, registry))
		protected.GET("/partials/entries/{id}", handlers.ListEntries(app, registry))
		protected.GET("/exhibitors/new", handlers.NewExhibitorForm(registry))
		protected.POST("/exhibitors/new", handlers.CreateExhibitor(app, registry))
		protected.GET("/exhibitors/{id}/edit", handlers.EditExhibitorForm(app, registry))
		protected.POST("/exhibitors/{id}/edit", handlers.UpdateExhibitor(app, registry))
		protected.DELETE("/exhibitors/{id}", handlers.DeleteExhibitor(app, registry))
		protected.GET("/exhibitors/{id}/entries", handlers.ShowEntries(app, registry))
		protected.POST("/exhibitors/{id}/entries", handlers.CreateEntry(app, registry))
		protected.DELETE("/exhibitors/{id}/entries/{entryid}", handlers.DeleteEntry(app, registry))

		se.Router.GET("/static/{path...}", apis.Static(os.DirFS("./pb_public"), false))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
