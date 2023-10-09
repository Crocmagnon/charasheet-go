package main

import (
	"net/http"

	"github.com/Crocmagnon/charasheet-go/assets"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	mux := httprouter.New()
	mux.NotFound = http.HandlerFunc(app.notFound)

	fileServer := http.FileServer(http.FS(assets.EmbeddedFiles))
	mux.Handler("GET", "/static/*filepath", fileServer)

	appMiddleware := alice.New(app.preventCSRF)

	mux.Handler("GET", "/version", appMiddleware.ThenFunc(app.version))

	appMiddleware = appMiddleware.Append(app.authenticate)
	mux.Handler("GET", "/", appMiddleware.ThenFunc(app.home))

	anonymous := appMiddleware.Append(app.requireAnonymousUser)
	mux.Handler("GET", "/signup", anonymous.ThenFunc(app.signup))
	mux.Handler("POST", "/signup", anonymous.ThenFunc(app.signup))
	mux.Handler("GET", "/login", anonymous.ThenFunc(app.login))
	mux.Handler("POST", "/login", anonymous.ThenFunc(app.login))
	mux.Handler("GET", "/forgotten-password", anonymous.ThenFunc(app.forgottenPassword))
	mux.Handler("POST", "/forgotten-password", anonymous.ThenFunc(app.forgottenPassword))
	mux.Handler("GET", "/forgotten-password-confirmation", anonymous.ThenFunc(app.forgottenPasswordConfirmation))
	mux.Handler("GET", "/password-reset/:plaintextToken", anonymous.ThenFunc(app.passwordReset))
	mux.Handler("POST", "/password-reset/:plaintextToken", anonymous.ThenFunc(app.passwordReset))
	mux.Handler("GET", "/password-reset-confirmation", anonymous.ThenFunc(app.passwordResetConfirmation))

	authenticated := appMiddleware.Append(app.requireAuthenticatedUser)
	mux.Handler("POST", "/logout", authenticated.ThenFunc(app.logout))

	mux.Handler("GET", "/character/:id/notes_change/", authenticated.ThenFunc(app.characterNotesChange))
	mux.Handler("POST", "/character/:id/notes_change/", authenticated.ThenFunc(app.characterNotesChange))

	defaultMiddleware := alice.New(app.logging, app.recoverPanic, app.securityHeaders)
	return defaultMiddleware.Then(mux)
}
