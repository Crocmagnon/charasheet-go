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

	appMiddleware := alice.New(app.preventCSRF, app.authenticate)

	mux.Handler("GET", "/", appMiddleware.ThenFunc(app.home))

	mux.Handler("GET", "/signup", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.signup))
	mux.Handler("POST", "/signup", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.signup))
	mux.Handler("GET", "/login", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.login))
	mux.Handler("POST", "/login", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.login))
	mux.Handler("GET", "/forgotten-password", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.forgottenPassword))
	mux.Handler("POST", "/forgotten-password", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.forgottenPassword))
	mux.Handler("GET", "/forgotten-password-confirmation", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.forgottenPasswordConfirmation))
	mux.Handler("GET", "/password-reset/:plaintextToken", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.passwordReset))
	mux.Handler("POST", "/password-reset/:plaintextToken", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.passwordReset))
	mux.Handler("GET", "/password-reset-confirmation", appMiddleware.Append(app.requireAnonymousUser).ThenFunc(app.passwordResetConfirmation))

	mux.Handler("POST", "/logout", appMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.logout))

	defaultMiddleware := alice.New(app.recoverPanic, app.securityHeaders)
	return defaultMiddleware.Then(mux)
}
