// Package handler provides HTTP handlers for the blockroll web application.
package handler

import (
	"context"
	"net/http"

	"github.com/zandenkane/blockroll/pkg/store"
)

type contextKey string

const userIDKey contextKey = "userID"

// SessionAuth is middleware that checks for a valid session cookie.
// If the cookie is missing or invalid, it redirects to the login page.
// If valid, it sets the user ID in the request context.
func SessionAuth(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			userID, err := s.GetSession(cookie.Value)
			if err != nil || userID == "" {
				// Invalid or expired session, clear the cookie and redirect.
				http.SetCookie(w, &http.Cookie{
					Name:     "session",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the authenticated user's ID from the request context.
// Returns "" if not authenticated (should not happen behind SessionAuth middleware).
func GetUserID(r *http.Request) string {
	v, _ := r.Context().Value(userIDKey).(string)
	return v
}
