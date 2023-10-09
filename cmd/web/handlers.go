package main

import (
	"net/http"
	"time"

	"github.com/Crocmagnon/charasheet-go/internal/password"
	"github.com/Crocmagnon/charasheet-go/internal/request"
	"github.com/Crocmagnon/charasheet-go/internal/response"
	"github.com/Crocmagnon/charasheet-go/internal/token"
	"github.com/Crocmagnon/charasheet-go/internal/validator"

	"github.com/julienschmidt/httprouter"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	err := response.Page(w, http.StatusOK, data, "pages/home.tmpl")
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) signup(w http.ResponseWriter, r *http.Request) {
	var form struct {
		Email     string              `form:"Email"`
		Password  string              `form:"Password"`
		Validator validator.Validator `form:"-"`
	}

	switch r.Method {
	case http.MethodGet:
		data := app.newTemplateData(r)
		data["Form"] = form

		err := response.Page(w, http.StatusOK, data, "pages/signup.tmpl")
		if err != nil {
			app.serverError(w, r, err)
		}

	case http.MethodPost:
		err := request.DecodePostForm(r, &form)
		if err != nil {
			app.badRequest(w, r, err)
			return
		}

		existingUser, err := app.db.GetUserByEmail(form.Email)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		form.Validator.CheckField(form.Email != "", "Email", "Email is required")
		form.Validator.CheckField(validator.Matches(form.Email, validator.RgxEmail), "Email", "Must be a valid email address")
		form.Validator.CheckField(existingUser == nil, "Email", "Email is already in use")

		form.Validator.CheckField(form.Password != "", "Password", "Password is required")
		form.Validator.CheckField(len(form.Password) >= 8, "Password", "Password is too short")
		form.Validator.CheckField(len(form.Password) <= 72, "Password", "Password is too long")
		form.Validator.CheckField(validator.NotIn(form.Password, password.CommonPasswords...), "Password", "Password is too common")

		if form.Validator.HasErrors() {
			data := app.newTemplateData(r)
			data["Form"] = form

			err := response.Page(w, http.StatusUnprocessableEntity, data, "pages/signup.tmpl")
			if err != nil {
				app.serverError(w, r, err)
			}
			return
		}

		hashedPassword, err := password.Hash(form.Password)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		id, err := app.db.InsertUser(form.Email, hashedPassword)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		session, err := app.sessionStore.Get(r, "session")
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		session.Values["userID"] = id

		err = session.Save(r, w)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {
	var form struct {
		Email     string              `form:"Email"`
		Password  string              `form:"Password"`
		Validator validator.Validator `form:"-"`
	}

	switch r.Method {
	case http.MethodGet:
		data := app.newTemplateData(r)
		data["Form"] = form

		err := response.Page(w, http.StatusOK, data, "pages/login.tmpl")
		if err != nil {
			app.serverError(w, r, err)
		}

	case http.MethodPost:
		err := request.DecodePostForm(r, &form)
		if err != nil {
			app.badRequest(w, r, err)
			return
		}

		user, err := app.db.GetUserByEmail(form.Email)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		form.Validator.CheckField(form.Email != "", "Email", "Email is required")
		form.Validator.CheckField(user != nil, "Email", "Email address could not be found")

		if user != nil {
			passwordMatches, err := password.Matches(form.Password, user.HashedPassword)
			if err != nil {
				app.serverError(w, r, err)
				return
			}

			form.Validator.CheckField(form.Password != "", "Password", "Password is required")
			form.Validator.CheckField(passwordMatches, "Password", "Password is incorrect")
		}

		if form.Validator.HasErrors() {
			data := app.newTemplateData(r)
			data["Form"] = form

			err := response.Page(w, http.StatusUnprocessableEntity, data, "pages/login.tmpl")
			if err != nil {
				app.serverError(w, r, err)
			}
			return
		}

		session, err := app.sessionStore.Get(r, "session")
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		session.Values["userID"] = user.ID

		redirectPath, ok := session.Values["redirectPathAfterLogin"].(string)
		if ok {
			delete(session.Values, "redirectPathAfterLogin")
		} else {
			redirectPath = "/"
		}

		err = session.Save(r, w)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		http.Redirect(w, r, redirectPath, http.StatusSeeOther)
	}
}

func (app *application) logout(w http.ResponseWriter, r *http.Request) {
	session, err := app.sessionStore.Get(r, "session")
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	delete(session.Values, "userID")

	err = session.Save(r, w)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) forgottenPassword(w http.ResponseWriter, r *http.Request) {
	var form struct {
		Email     string              `form:"Email"`
		Validator validator.Validator `form:"-"`
	}

	switch r.Method {
	case http.MethodGet:
		data := app.newTemplateData(r)
		data["Form"] = form

		err := response.Page(w, http.StatusOK, data, "pages/forgotten-password.tmpl")
		if err != nil {
			app.serverError(w, r, err)
		}

	case http.MethodPost:
		err := request.DecodePostForm(r, &form)
		if err != nil {
			app.badRequest(w, r, err)
			return
		}

		user, err := app.db.GetUserByEmail(form.Email)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		form.Validator.CheckField(form.Email != "", "Email", "Email is required")
		form.Validator.CheckField(validator.Matches(form.Email, validator.RgxEmail), "Email", "Must be a valid email address")
		form.Validator.CheckField(user != nil, "Email", "No matching email found")

		if form.Validator.HasErrors() {
			data := app.newTemplateData(r)
			data["Form"] = form

			err := response.Page(w, http.StatusUnprocessableEntity, data, "pages/forgotten-password.tmpl")
			if err != nil {
				app.serverError(w, r, err)
			}
			return
		}

		plaintextToken, err := token.New()
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		hashedToken := token.Hash(plaintextToken)

		err = app.db.InsertPasswordReset(hashedToken, user.ID, 24*time.Hour)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		data := app.newEmailData()
		data["PlaintextToken"] = plaintextToken

		err = app.mailer.Send(user.Email, data, "forgotten-password.tmpl")
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		http.Redirect(w, r, "/forgotten-password-confirmation", http.StatusSeeOther)
	}
}

func (app *application) forgottenPasswordConfirmation(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	err := response.Page(w, http.StatusOK, data, "pages/forgotten-password-confirmation.tmpl")
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) passwordReset(w http.ResponseWriter, r *http.Request) {
	plaintextToken := httprouter.ParamsFromContext(r.Context()).ByName("plaintextToken")

	hashedToken := token.Hash(plaintextToken)

	passwordReset, err := app.db.GetPasswordReset(hashedToken)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if passwordReset == nil {
		data := app.newTemplateData(r)
		data["InvalidLink"] = true

		err := response.Page(w, http.StatusUnprocessableEntity, data, "pages/password-reset.tmpl")
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	var form struct {
		NewPassword string              `form:"NewPassword"`
		Validator   validator.Validator `form:"-"`
	}

	switch r.Method {
	case http.MethodGet:
		data := app.newTemplateData(r)
		data["Form"] = form
		data["PlaintextToken"] = plaintextToken

		err := response.Page(w, http.StatusOK, data, "pages/password-reset.tmpl")
		if err != nil {
			app.serverError(w, r, err)
		}

	case http.MethodPost:
		err := request.DecodePostForm(r, &form)
		if err != nil {
			app.badRequest(w, r, err)
			return
		}

		form.Validator.CheckField(form.NewPassword != "", "NewPassword", "New password is required")
		form.Validator.CheckField(len(form.NewPassword) >= 8, "NewPassword", "New password is too short")
		form.Validator.CheckField(len(form.NewPassword) <= 72, "NewPassword", "New password is too long")
		form.Validator.CheckField(validator.NotIn(form.NewPassword, password.CommonPasswords...), "NewPassword", "New password is too common")

		if form.Validator.HasErrors() {
			data := app.newTemplateData(r)
			data["Form"] = form
			data["PlaintextToken"] = plaintextToken

			err := response.Page(w, http.StatusUnprocessableEntity, data, "pages/password-reset.tmpl")
			if err != nil {
				app.serverError(w, r, err)
			}
			return
		}

		hashedPassword, err := password.Hash(form.NewPassword)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		err = app.db.UpdateUserHashedPassword(passwordReset.UserID, hashedPassword)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		err = app.db.DeletePasswordResets(passwordReset.UserID)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		http.Redirect(w, r, "/password-reset-confirmation", http.StatusSeeOther)
	}
}

func (app *application) passwordResetConfirmation(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	err := response.Page(w, http.StatusOK, data, "pages/password-reset-confirmation.tmpl")
	if err != nil {
		app.serverError(w, r, err)
	}
}
