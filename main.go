package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
)

func main() {
	app := pocketbase.New()

	// Register the JSVM plugin
	jsvm.MustRegister(app, jsvm.Config{
		// You can add configurations here if needed,
		// such as custom Go bindings or an OnInit function.
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// serves static files from the provided public dir (if exists)
		se.Router.GET("/*", apis.Static(os.DirFS("./pb_public"), false))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
