package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/zandenkane/blockroll/pkg/privacy"
	"github.com/zandenkane/blockroll/pkg/store"
)

// TrustHandler manages trust tier assignments between users.
type TrustHandler struct {
	Store     *store.Store
	Templates *template.Template
}

// trustData holds the template data for the trust management page.
type trustData struct {
	NeighborhoodID   string
	NeighborhoodName string
	Members          []memberWithTrust
	Error            string
	Success          string
}

// memberWithTrust pairs a neighborhood member with the trust tier the current user grants them.
type memberWithTrust struct {
	ID       string
	Email    string
	Tier     int
	TierName string
}

// Index shows the trust management page for a neighborhood.
func (h *TrustHandler) Index(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	nID := chi.URLParam(r, "neighborhoodID")

	member, err := h.Store.IsNeighborhoodMember(nID, userID)
	if err != nil || !member {
		http.Redirect(w, r, "/neighborhoods", http.StatusSeeOther)
		return
	}

	data, err := h.buildTrustData(userID, nID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.Templates.ExecuteTemplate(w, "trust.html", data)
}

// SetTrust processes a trust tier update form submission.
func (h *TrustHandler) SetTrust(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	nID := chi.URLParam(r, "neighborhoodID")

	member, err := h.Store.IsNeighborhoodMember(nID, userID)
	if err != nil || !member {
		http.Redirect(w, r, "/neighborhoods", http.StatusSeeOther)
		return
	}

	neighborID := r.FormValue("neighbor_id")
	tierStr := r.FormValue("tier")

	if neighborID == "" || neighborID == userID {
		data, _ := h.buildTrustData(userID, nID)
		data.Error = "Invalid neighbor selection."
		h.Templates.ExecuteTemplate(w, "trust.html", data)
		return
	}

	tier, err := strconv.Atoi(tierStr)
	if err != nil || !privacy.ValidTier(tier) {
		data, _ := h.buildTrustData(userID, nID)
		data.Error = "Invalid trust tier (must be 0 through 3)."
		h.Templates.ExecuteTemplate(w, "trust.html", data)
		return
	}

	// Verify the neighbor is actually in this neighborhood.
	neighborMember, err := h.Store.IsNeighborhoodMember(nID, neighborID)
	if err != nil || !neighborMember {
		data, _ := h.buildTrustData(userID, nID)
		data.Error = "That user is not a member of this neighborhood."
		h.Templates.ExecuteTemplate(w, "trust.html", data)
		return
	}

	if err := h.Store.SetTrust(userID, neighborID, tier); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data, _ := h.buildTrustData(userID, nID)
	data.Success = "Trust updated."
	h.Templates.ExecuteTemplate(w, "trust.html", data)
}

func (h *TrustHandler) buildTrustData(userID, nID string) (*trustData, error) {
	n, err := h.Store.GetNeighborhoodByID(nID)
	if err != nil || n == nil {
		return &trustData{NeighborhoodID: nID}, err
	}

	members, err := h.Store.GetNeighborhoodMembers(nID)
	if err != nil {
		return &trustData{NeighborhoodID: nID, NeighborhoodName: n.Name}, err
	}

	var mwt []memberWithTrust
	for _, m := range members {
		if m.ID == userID {
			continue // skip self
		}
		tier, err := h.Store.GetTrust(userID, m.ID)
		if err != nil {
			tier = privacy.TierNone
		}
		mwt = append(mwt, memberWithTrust{
			ID:       m.ID,
			Email:    m.Email,
			Tier:     tier,
			TierName: privacy.TierName(tier),
		})
	}

	return &trustData{
		NeighborhoodID:   nID,
		NeighborhoodName: n.Name,
		Members:          mwt,
	}, nil
}
