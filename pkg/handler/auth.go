package handler

import (
	"html/template"
	"net/http"

	"github.com/zandenkane/blockroll/pkg/store"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles registration, login, and logout.
type AuthHandler struct {
	Store     *store.Store
	Templates *template.Template
}

// RegisterForm renders the registration page.
func (h *AuthHandler) RegisterForm(w http.ResponseWriter, r *http.Request) {
	h.Templates.ExecuteTemplate(w, "register.html", nil)
}

// Register processes a registration form submission.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.Templates.ExecuteTemplate(w, "register.html", map[string]string{
			"Error": "Email and password are required.",
		})
		return
	}

	if len(password) < 8 {
		h.Templates.ExecuteTemplate(w, "register.html", map[string]string{
			"Error": "Password must be at least 8 characters.",
		})
		return
	}

	// Check if email already exists.
	existing, err := h.Store.GetUserByEmail(email)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		h.Templates.ExecuteTemplate(w, "register.html", map[string]string{
			"Error": "An account with that email already exists.",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	userID, err := h.Store.CreateUser(email, string(hash))
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Log in automatically after registration.
	token, err := h.Store.CreateSession(userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LoginForm renders the login page.
func (h *AuthHandler) LoginForm(w http.ResponseWriter, r *http.Request) {
	h.Templates.ExecuteTemplate(w, "login.html", nil)
}

// Login processes a login form submission.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.Templates.ExecuteTemplate(w, "login.html", map[string]string{
			"Error": "Email and password are required.",
		})
		return
	}

	user, err := h.Store.GetUserByEmail(email)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		h.Templates.ExecuteTemplate(w, "login.html", map[string]string{
			"Error": "Invalid email or password.",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		h.Templates.ExecuteTemplate(w, "login.html", map[string]string{
			"Error": "Invalid email or password.",
		})
		return
	}

	token, err := h.Store.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout clears the session and redirects to login.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.Store.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
