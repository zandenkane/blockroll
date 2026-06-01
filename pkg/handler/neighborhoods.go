package handler

import (
	"html/template"
	"net/http"

	"github.com/zandenkane/blockroll/pkg/store"
)

// NeighborhoodHandler manages neighborhood creation and joining.
type NeighborhoodHandler struct {
	Store     *store.Store
	Templates *template.Template
}

// Form renders the neighborhood management page.
func (h *NeighborhoodHandler) Form(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	neighborhoods, err := h.Store.GetUserNeighborhoods(userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.Templates.ExecuteTemplate(w, "neighborhood.html", map[string]any{
		"Neighborhoods": neighborhoods,
	})
}

// Create processes a "create neighborhood" form submission.
func (h *NeighborhoodHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	name := r.FormValue("name")

	if name == "" {
		h.renderWithError(w, userID, "Neighborhood name is required.")
		return
	}

	n, err := h.Store.CreateNeighborhood(name, userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	neighborhoods, _ := h.Store.GetUserNeighborhoods(userID)
	h.Templates.ExecuteTemplate(w, "neighborhood.html", map[string]any{
		"Neighborhoods": neighborhoods,
		"Success":       "Created neighborhood \"" + n.Name + "\". Share the join code: " + n.JoinCode,
	})
}

// Join processes a "join neighborhood" form submission.
func (h *NeighborhoodHandler) Join(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	code := r.FormValue("code")

	if code == "" {
		h.renderWithError(w, userID, "Join code is required.")
		return
	}

	n, err := h.Store.JoinNeighborhood(code, userID)
	if err != nil {
		h.renderWithError(w, userID, "Invalid join code.")
		return
	}

	neighborhoods, _ := h.Store.GetUserNeighborhoods(userID)
	h.Templates.ExecuteTemplate(w, "neighborhood.html", map[string]any{
		"Neighborhoods": neighborhoods,
		"Success":       "Joined neighborhood \"" + n.Name + "\".",
	})
}

func (h *NeighborhoodHandler) renderWithError(w http.ResponseWriter, userID, errMsg string) {
	neighborhoods, _ := h.Store.GetUserNeighborhoods(userID)
	h.Templates.ExecuteTemplate(w, "neighborhood.html", map[string]any{
		"Neighborhoods": neighborhoods,
		"Error":         errMsg,
	})
}
