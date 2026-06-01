package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/zandenkane/blockroll/pkg/privacy"
	"github.com/zandenkane/blockroll/pkg/store"
)

// ListingHandler manages CRUD operations for skill/resource listings.
type ListingHandler struct {
	Store     *store.Store
	Templates *template.Template
}

// listingsData holds the template data for the listings page.
type listingsData struct {
	NeighborhoodID   string
	NeighborhoodName string
	Listings         []store.Listing
	MyListings       []store.Listing
	Error            string
	Success          string
}

// Index shows all visible listings in a neighborhood.
func (h *ListingHandler) Index(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	nID := chi.URLParam(r, "neighborhoodID")

	// Verify membership.
	member, err := h.Store.IsNeighborhoodMember(nID, userID)
	if err != nil || !member {
		http.Redirect(w, r, "/neighborhoods", http.StatusSeeOther)
		return
	}

	n, err := h.Store.GetNeighborhoodByID(nID)
	if err != nil || n == nil {
		http.Redirect(w, r, "/neighborhoods", http.StatusSeeOther)
		return
	}

	listings, err := h.Store.GetVisibleListings(nID, userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	myListings, err := h.Store.GetUserListings(userID, nID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.Templates.ExecuteTemplate(w, "listings.html", listingsData{
		NeighborhoodID:   nID,
		NeighborhoodName: n.Name,
		Listings:         listings,
		MyListings:       myListings,
	})
}

// Create processes a new listing form submission.
func (h *ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	nID := chi.URLParam(r, "neighborhoodID")

	member, err := h.Store.IsNeighborhoodMember(nID, userID)
	if err != nil || !member {
		http.Redirect(w, r, "/neighborhoods", http.StatusSeeOther)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	category := r.FormValue("category")
	minTierStr := r.FormValue("min_tier")

	if title == "" {
		h.renderError(w, userID, nID, "Title is required.")
		return
	}

	minTier, err := strconv.Atoi(minTierStr)
	if err != nil || !privacy.ValidTier(minTier) {
		h.renderError(w, userID, nID, "Invalid visibility tier.")
		return
	}

	if category == "" {
		category = "other"
	}

	_, err = h.Store.CreateListing(userID, nID, title, description, category, minTier)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/neighborhoods/"+nID+"/listings", http.StatusSeeOther)
}

// Edit processes a listing edit form submission.
func (h *ListingHandler) Edit(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	nID := chi.URLParam(r, "neighborhoodID")
	listingID := chi.URLParam(r, "listingID")

	listing, err := h.Store.GetListingByID(listingID)
	if err != nil || listing == nil || listing.UserID != userID {
		http.Redirect(w, r, "/neighborhoods/"+nID+"/listings", http.StatusSeeOther)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	category := r.FormValue("category")
	minTierStr := r.FormValue("min_tier")

	if title == "" {
		title = listing.Title
	}
	if category == "" {
		category = listing.Category
	}

	minTier, err := strconv.Atoi(minTierStr)
	if err != nil || !privacy.ValidTier(minTier) {
		minTier = listing.MinTier
	}

	if err := h.Store.UpdateListing(listingID, title, description, category, minTier); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/neighborhoods/"+nID+"/listings", http.StatusSeeOther)
}

// Delete removes a listing owned by the current user.
func (h *ListingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	nID := chi.URLParam(r, "neighborhoodID")
	listingID := chi.URLParam(r, "listingID")

	listing, err := h.Store.GetListingByID(listingID)
	if err != nil || listing == nil || listing.UserID != userID {
		http.Redirect(w, r, "/neighborhoods/"+nID+"/listings", http.StatusSeeOther)
		return
	}

	h.Store.DeleteListing(listingID)
	http.Redirect(w, r, "/neighborhoods/"+nID+"/listings", http.StatusSeeOther)
}

func (h *ListingHandler) renderError(w http.ResponseWriter, userID, nID, errMsg string) {
	n, _ := h.Store.GetNeighborhoodByID(nID)
	name := ""
	if n != nil {
		name = n.Name
	}
	listings, _ := h.Store.GetVisibleListings(nID, userID)
	myListings, _ := h.Store.GetUserListings(userID, nID)
	h.Templates.ExecuteTemplate(w, "listings.html", listingsData{
		NeighborhoodID:   nID,
		NeighborhoodName: name,
		Listings:         listings,
		MyListings:       myListings,
		Error:            errMsg,
	})
}
