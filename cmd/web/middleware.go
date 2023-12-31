package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/justinas/nosurf"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")

		next.ServeHTTP(w, r)
	})
}

func (app *application) preventCSRF(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	return csrfHandler
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := app.sessionStore.Get(r, "session")
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		userID, ok := session.Values["userID"].(int)
		if !ok {
			userID, err = app.getUserIDFromDjangoSession(r)
			if err != nil {
				app.serverError(w, r, err)
				return
			}

			ok = userID > 0
		}

		if ok {
			user, err := app.db.GetUser(userID)
			if err != nil {
				app.serverError(w, r, err)
				return
			}

			if user != nil {
				r = contextSetAuthenticatedUser(r, user)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) getUserIDFromDjangoSession(r *http.Request) (int, error) {
	sessionIDCookie, err := r.Cookie("sessionid")

	switch {
	case errors.Is(err, http.ErrNoCookie):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("getting cookie 'sessionid': %w", err)
	}

	session, err := app.db.GetSession(sessionIDCookie.Value)
	if err != nil {
		return 0, fmt.Errorf("getting session from db: %w", err)
	}

	sessionData, err := session.Decode()
	if err != nil {
		return 0, fmt.Errorf("decoding session: %w", err)
	}

	userID, err := strconv.Atoi(sessionData.AuthUserID)
	if err != nil {
		return 0, fmt.Errorf("converting userID to int: %w", err)
	}

	return userID, nil
}

func (app *application) requireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticatedUser := contextGetAuthenticatedUser(r)

		if authenticatedUser == nil {
			session, err := app.sessionStore.Get(r, "session")
			if err != nil {
				app.serverError(w, r, err)
				return
			}

			session.Values["redirectPathAfterLogin"] = r.URL.Path

			err = session.Save(r, w)
			if err != nil {
				app.serverError(w, r, err)
				return
			}

			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		w.Header().Add("Cache-Control", "no-store")

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAnonymousUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticatedUser := contextGetAuthenticatedUser(r)

		if authenticatedUser != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &StatusRecorder{
			ResponseWriter: w,
			Status:         http.StatusOK,
		}
		next.ServeHTTP(recorder, r)
		app.logger.Info("processed request", slog.Group("http", "status", recorder.Status, "method", r.Method, "path", r.URL.Path))
	})
}

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}
