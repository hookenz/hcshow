package middleware

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RequireAuth(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		cookie, err := e.Request.Cookie("pb_auth")
		if err != nil {
			return redirectToLogin(e)
		}

		record, err := app.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
		if err != nil {
			http.SetCookie(e.Response, &http.Cookie{
				Name:   "pb_auth",
				Value:  "",
				MaxAge: -1,
				Path:   "/",
			})
			return redirectToLogin(e)
		}

		e.Auth = record
		return e.Next()
	}
}

func redirectToLogin(e *core.RequestEvent) error {
	if e.Request.Header.Get("HX-Request") == "true" {
		e.Response.Header().Set("HX-Redirect", "/login")
		return e.NoContent(401)
	}
	return e.Redirect(307, "/login")
}
